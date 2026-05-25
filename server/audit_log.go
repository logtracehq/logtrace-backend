package server

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"gitlab.com/logtrace/logtrace"
	queue "gitlab.com/logtrace/logtrace/internal/pkg/queues"
	"gitlab.com/logtrace/logtrace/internal/pkg/util"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type auditLogHandler struct {
	auditLogRepo logtrace.AuditLogRepository
	userRepo     logtrace.UserRepository
	orgRepo      logtrace.OrganizationRepository
	orgUserRepo  logtrace.OrganizationUserRepository
	queue        queue.QueueHandler
}

type createAuditLogRequest struct {
	GenericRequest

	Action          string            `json:"action"`
	Timestamp       string            `json:"timestamp"`
	IPAddress       string            `json:"ip_address"`
	UserID          string            `json:"user_id"`
	UserName        string            `json:"username"`
	RequestID       string            `json:"request_id"`
	Client          string            `json:"client"`
	OperatingSystem string            `json:"operating_system"`
	OrganizationID  string            `json:"organization_id"`
	Metadata        logtrace.Metadata `json:"metadata" `
}

func (a *createAuditLogRequest) Validate() error {
	if util.IsStringEmpty(a.Action) {
		return errors.New("action is required")
	}
	if util.IsStringEmpty(a.Timestamp) {
		return errors.New("timestamp is required")
	}
	if util.IsStringEmpty(a.UserID) && util.IsStringEmpty(a.UserName) {
		return errors.New("at least one of user_id or username is required")
	}
	return nil
}

// @Description Create a new audit log entry
// @Tags Audit Logs
// @Accept json
// @Produce json
// @Param auditLog body createAuditLogRequest true "Audit log creation request"
// @Success 201 {object} APIStatus "Audit log created successfully"
// @Failure 400 {object} APIStatus "Invalid request body"
// @Failure 500 {object} APIStatus "Could not create audit log at this time. an error occurred"
// @Router /v1/audit-logs [post]
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

	auditLog := &logtrace.AuditLog{
		Action:          req.Action,
		IPAddress:       req.IPAddress,
		Timestamp:       time.Now().UTC(),
		OrganizationID:  getOrganizationFromContext(r.Context()).ID,
		UserID:          req.UserID,
		Metadata:        req.Metadata,
		RequestID:       req.RequestID,
		Client:          req.Client,
		OperatingSystem: req.OperatingSystem,
	}

	orgUser, err := a.orgUserRepo.Find(ctx, &logtrace.FindOrganizationUserOptions{
		UserID:         auditLog.UserID,
		OrganizationID: auditLog.OrganizationID,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logger.Error("failed to find organization user", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to find organization user"), StatusFailed
	}

	if orgUser != nil {
		req.UserID = orgUser.UserID
		req.UserName = orgUser.Username
	} else {
		_, err := a.orgUserRepo.Create(ctx, &logtrace.OrganizationUser{
			UserID:         req.UserID,
			OrganizationID: auditLog.OrganizationID,
			Username:       req.UserName,
		})
		if err != nil {
			logger.Error("failed to create organization user", zap.Error(err))
			return newAPIStatus(http.StatusInternalServerError, "failed to create organization user"), StatusFailed
		}
	}

	auditLog.UserID = req.UserID
	auditLog.UserName = req.UserName

	if err := a.queue.Add(ctx, queue.QueueTopicSaveAuditLog, queue.SaveAuditLogOptions{AuditLog: auditLog}); err != nil {
		logger.Error("failed to enqueue audit log", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to enqueue audit log"), StatusFailed
	}

	return newAPIStatus(http.StatusAccepted, "Audit log accepted for processing"), StatusSuccess
}

// @Description Get a specific audit log entry by ID
// @Tags Audit Logs
// @Accept json
// @Produce json
// @Param reference path string true "Audit log ID"
// @Success 200 {object} listAuditLog "Audit log fetched successfully"
// @Failure 400 {object} APIStatus "Invalid audit log reference"
// @Failure 404 {object} APIStatus "Audit log not found"
// @Failure 500 {object} APIStatus "Failed to fetch audit log"
// @Router /v1/audit-logs/{reference} [get]
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

	opts := &logtrace.FindAuditLogOptions{
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
		ID:        auditLog.ID,
		Action:    auditLog.Action,
		UserName:  auditLog.UserName,
		Timestamp: auditLog.Timestamp,
		IPAddress: auditLog.IPAddress,
		UserID:    auditLog.UserID,
		Metadata:  auditLog.Metadata,
		CreatedAt: auditLog.CreatedAt,
		RequestID: auditLog.RequestID,
	}

	return listAuditLog{
		AuditLog:  auditLogResponse,
		APIStatus: newAPIStatus(http.StatusOK, "Audit log fetched successfully"),
	}, StatusSuccess
}

// @Description Get a list of audit log entries with pagination and filtering
// @Tags Audit Logs
// @Accept json
// @Produce json
// @Param search query string false "Search term for filtering audit logs"
// @Param start_date query string false "Start date for filtering audit logs (ISO 8601 format)"
// @Param end_date query string false "End date for filtering audit logs (ISO 8601 format)"
// @Param page query int false "Page number for pagination"
// @Param per_page query int false "Number of items per page for pagination"
// @Success 200 {object} listAllAuditLogs "Audit logs fetched successfully"
// @Failure 400 {object} APIStatus "Invalid query parameters"
// @Failure 500 {object} APIStatus "Failed to fetch audit logs"
// @Router /v1/audit-logs [get]
func (a *auditLogHandler) ListAll(ctx context.Context, span trace.Span, logger *zap.Logger, w http.ResponseWriter,
	r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("listing all audit logs")

	query := r.URL.Query()
	search := query.Get("search")
	startDate := query.Get("start_date")
	endDate := query.Get("end_date")
	client := query.Get("client")
	operatingSystem := query.Get("operating_system")

	opts := logtrace.FindAuditLogOptions{
		OrganizationID:  getOrganizationFromContext(r.Context()).ID,
		Paginator:       logtrace.PaginatorFromRequest(r),
		Search:          search,
		EndDate:         endDate,
		StartDate:       startDate,
		Client:          client,
		OperatingSystem: operatingSystem,
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
			ID:              a.ID,
			Action:          a.Action,
			UserName:        a.UserName,
			Timestamp:       a.Timestamp,
			IPAddress:       a.IPAddress,
			UserID:          a.UserID,
			Metadata:        a.Metadata,
			RequestID:       a.RequestID,
			CreatedAt:       a.CreatedAt,
			Client:          a.Client,
			OperatingSystem: a.OperatingSystem,
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

// @Description Get audit log metrics such as total count of audit logs
// @Tags Audit Logs
// @Accept json
// @Produce json
// @Success 200 {object} auditLogMetrics "Audit log metrics fetched successfully"
// @Failure 500 {object} APIStatus "Failed to fetch metrics"
// @Router /v1/audit-logs/metrics [get]
func (a *auditLogHandler) Metrics(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("fetching audit log metrics")

	opts := logtrace.FindAuditLogOptions{
		OrganizationID: getOrganizationFromContext(r.Context()).ID,
	}

	totalCount, err := a.auditLogRepo.Metrics(ctx, &opts)
	if err != nil {
		logger.Error("failed to fetch audit log metrics", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to fetch metrics"), StatusFailed
	}

	return auditLogMetrics{
		Count:     totalCount,
		APIStatus: newAPIStatus(http.StatusOK, "Audit log metrics fetched successfully"),
	}, StatusSuccess
}
