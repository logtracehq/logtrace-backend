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
	ResourceID     string            `json:"resource_id"`
	IPAddress      string            `json:"ip_address"`
	UserID         string            `json:"user_id"`
	RequestID      string            `json:"request_id"`
	OrganizationID string            `json:"organization_id"`
	MetaData       *logbase.MetaData `json:"metadata" `
}

func (a *createAuditLogRequest) Validate() error {
	if util.IsStringEmpty(a.Action) {
		return errors.New("action is required")
	}
	if util.IsStringEmpty(a.Timestamp) {
		return errors.New("timestamp is required")
	}
	if util.IsStringEmpty(a.ResourceID) {
		return errors.New("resource id is required")
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

	resourceUUID, err := uuid.Parse(req.ResourceID)
	if err != nil {
		logger.Error("invalid resource ID", zap.Error(err))
		return newAPIStatus(http.StatusBadRequest, "invalid resource ID"), StatusFailed
	}

	auditLog := &logbase.AuditLog{
		Action:         req.Action,
		ResourceID:     resourceUUID,
		IPAddress:      req.IPAddress,
		Timestamp:      time.Now().UTC(),
		OrganizationID: getOrganizationFromContext(r.Context()).ID,
		UserID:         getUserFromContext(r.Context()).ID,
		MetaData:       req.MetaData,
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

	return listAuditLog{
		AuditLog:  auditLog,
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

	return listAllAuditLogs{
		AuditLogs: auditLogs,
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
