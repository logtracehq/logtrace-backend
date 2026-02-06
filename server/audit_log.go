package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"gitlab.com/logbase/logbase"
	"gitlab.com/logbase/logbase/internal/pkg/util"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type auditLogHandler struct {
	auditLogRepo logbase.AuditLogRepository
	userRepo     logbase.UserRepository
	orgRepo      logbase.OrganizationRepository
}

type createAuditLogRequest struct {
	GenericRequest

	Action         string            `json:"action"`
	Timestamp      string            `json:"timestamp"`
	ProjectID      string            `json:"project_id"`
	IPAddress      string            `json:"ip_address"`
	UserID         string            `json:"user_id"`
	RequestID      string            `json:"request_id"`
	OrganizationID string            `json:"organization_id"`
	Metadata       *logbase.Metadata `json:"metadata" `
}

func (a *createAuditLogRequest) Validate() error {
	if util.IsStringEmpty(a.Action) {
		return errors.New("action is required")
	}
	if util.IsStringEmpty(a.Timestamp) {
		return errors.New("timestamp is required")
	}
	if util.IsStringEmpty(a.ProjectID) {
		return errors.New("project id is required")
	}

	return nil
}

func (a *auditLogHandler) Create(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("creating an audit log")
	span.SetAttributes(attribute.String("handler", "createAuditLog"))

	req := new(createAuditLogRequest)
	if err := render.Bind(r, req); err != nil {
		logger.Error("failed to bind request", zap.Error(err))
		return newAPIStatus(http.StatusBadRequest, "invalid request payload"), StatusFailed
	}

	if err := req.Validate(); err != nil {
		logger.Error("validation error", zap.Error(err))
		return newAPIStatus(http.StatusBadRequest, "failed to process request"), StatusFailed
	}

	projectUUID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		logger.Error("invalid project ID", zap.Error(err))
		return newAPIStatus(http.StatusBadRequest, "invalid project ID"), StatusFailed
	}

	auditLog := &logbase.AuditLog{
		Action:         req.Action,
		ProjectID:      projectUUID,
		IPAddress:      req.IPAddress,
		Timestamp:      time.Now().UTC(),
		OrganizationID: getOrganizationFromContext(r.Context()).ID,
		UserID:         getUserFromContext(r.Context()).ID,
		Metadata:       req.Metadata,
		RequestID:      req.RequestID,
	}

	err = a.auditLogRepo.Create(ctx, auditLog)
	if err != nil {
		logger.Error("failed to create audit log", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to create audit log"), StatusFailed
	}

	return newAPIStatus(http.StatusCreated, "Audit log created successfully"), StatusSuccess
}

func (a *auditLogHandler) List(ctx context.Context, span trace.Span, logger *zap.Logger, w http.ResponseWriter,
	r *http.Request,
) (render.Renderer, Status) {
	ref := chi.URLParam(r, "reference")
	span.SetAttributes(attribute.String("reference", ref))

	auditLogID, err := uuid.Parse(ref)
	if err != nil {
		logger.Error("invalid audit log id", zap.Error(err))
		return newAPIStatus(http.StatusBadRequest, "invalid audit log reference"), StatusFailed
	}

	opts := &logbase.FindAuditLogOptions{
		OrganizationID: getOrganizationFromContext(r.Context()).ID,
		ID:             auditLogID,
	}

	auditLog, err := a.auditLogRepo.List(ctx, opts)
	if err != nil {
		logger.Error("failed to fetch audit log", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to fetch audit log"), StatusFailed
	}
	if auditLog == nil {
		return newAPIStatus(http.StatusNotFound, "audit log not found"), StatusFailed
	}

	auditLogResponse := &AuditLog{
		ID:             auditLog.ID,
		Action:         auditLog.Action,
		Timestamp:      auditLog.Timestamp,
		ProjectID:      auditLog.ProjectID,
		IPAddress:      auditLog.IPAddress,
		UserID:         auditLog.UserID,
		Metadata:       auditLog.Metadata,
		CreatedAt:      auditLog.CreatedAt,
		RequestID:      auditLog.RequestID,
		OrganizationID: auditLog.OrganizationID,
		ProjectName:    auditLog.Project.Name,
		ProjectType:    auditLog.Project.Type,
	}

	return listAuditLog{
		AuditLog:  auditLogResponse,
		APIStatus: newAPIStatus(http.StatusOK, "Audit log fetched successfully"),
	}, StatusSuccess
}

func (a *auditLogHandler) ListAll(ctx context.Context, span trace.Span, logger *zap.Logger, w http.ResponseWriter,
	r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("listing all audit logs")

	opts := logbase.FindAuditLogOptions{
		OrganizationID: getOrganizationFromContext(r.Context()).ID,
		Paginator:      logbase.PaginatorFromRequest(r),
	}

	span.SetAttributes(opts.Paginator.OTELAttributes()...)

	auditLogs, count, err := a.auditLogRepo.ListAll(ctx, opts)
	if err != nil {
		logger.Error("failed to list audit logs", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to list audit logs"), StatusFailed
	}

	auditLogResponse := make([]*AuditLog, 0, len(auditLogs))
	for _, a := range auditLogs {
		dto := &AuditLog{
			ID:             a.ID,
			Action:         a.Action,
			Timestamp:      a.Timestamp,
			ProjectID:      a.ProjectID,
			IPAddress:      a.IPAddress,
			UserID:         a.UserID,
			Metadata:       a.Metadata,
			RequestID:      a.RequestID,
			OrganizationID: a.OrganizationID,
			CreatedAt:      a.CreatedAt,
			ProjectName:    a.Project.Name,
			ProjectType:    a.Project.Type,
		}

		auditLogResponse = append(auditLogResponse, dto)
	}

	return listAllAuditLogs{
		AuditLogs: auditLogResponse,
		Meta: meta{
			Paging: pagingInfo{
				Total:   count,
				Page:    opts.Paginator.Page,
				PerPage: opts.Paginator.PerPage,
			},
		},
		APIStatus: newAPIStatus(http.StatusOK, "Audit logs fetched successfully"),
	}, StatusSuccess
}
