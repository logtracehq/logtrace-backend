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
	"gitlab.com/logbase/logbase"
	"gitlab.com/logbase/logbase/config"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type sessionHandler struct {
	cfg         config.Config
	userRepo    logbase.UserRepository
	sessionRepo logbase.SessionRepository
	orgRepo     logbase.OrganizationRepository
	orgUserRepo logbase.OrganizationUserRepository
}

const (
	SessionStatusActive   = "ACTIVE"
	SessionStatusInactive = "INACTIVE"
)

type sessionRequest struct {
	GenericRequest

	LoginAt    string `json:"login_at"`
	DeviceInfo string `json:"device_info"`
	IPAddress  string `json:"ip_address"`
	Location   string `json:"location"`
	Status     string `json:"status"`
	UserID     string `json:"user_id"`
	UserName   string `json:"username"`
}

func (sr *sessionRequest) Validate() error {
	if sr.Status != SessionStatusActive && sr.Status != SessionStatusInactive {
		return errors.New("invalid status")
	}
	return nil
}

func (sh *sessionHandler) Create(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("creating the session ")

	req := new(sessionRequest)
	if err := render.Bind(r, req); err != nil {
		logger.Error("failed to bind create session request", zap.Error(err))
		return newAPIStatus(http.StatusBadRequest, "invalid request payload"), StatusFailed
	}

	if err := req.Validate(); err != nil {
		logger.Error("create session request validation failed", zap.Error(err))
		return newAPIStatus(http.StatusBadRequest, err.Error()), StatusFailed
	}

	loginAt, err := time.Parse("2006-01-02T15:04:05Z07:00", req.LoginAt)
	if err != nil {
		logger.Error("invalid login at format", zap.Error(err))
		return newAPIStatus(http.StatusBadRequest, "invalid login at format"), StatusFailed
	}

	session := &logbase.Session{
		UserID:         req.UserID,
		DeviceInfo:     req.DeviceInfo,
		UserName:       req.UserName,
		IPAddress:      req.IPAddress,
		Location:       req.Location,
		OrganizationID: getOrganizationFromContext(r.Context()).ID,
		Status:         req.Status,
		LoginAt:        loginAt,
	}

	orgUser, err := sh.orgUserRepo.Find(ctx, &logbase.FindOrganizationUserOptions{
		UserID:         session.UserID,
		OrganizationID: session.OrganizationID,
		Name:           session.UserName,
	})

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logger.Error("failed to find organization user", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to find organization user"), StatusFailed
	}

	if orgUser != nil {
		session.UserID = orgUser.UserID
		session.UserName = orgUser.Name
	} else {
		_, err := sh.orgUserRepo.Create(ctx, &logbase.OrganizationUser{
			UserID:         session.UserID,
			OrganizationID: session.OrganizationID,
			Name:           session.UserName,
		})
		if err != nil {
			logger.Error("failed to create organization user", zap.Error(err))
			return newAPIStatus(http.StatusInternalServerError, "failed to create organization user"), StatusFailed
		}
	}
	err = sh.sessionRepo.Create(ctx, session)
	if err != nil {
		logger.Error("an error occurred while creating session", zap.Error(err))
		return newAPIStatus(
			http.StatusInternalServerError,
			"could not create session at this time. an error occurred",
		), StatusFailed
	}

	return newAPIStatus(http.StatusCreated, "Session created successfully"), StatusSuccess
}

func (sh *sessionHandler) List(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	sessionID := chi.URLParam(r, "reference")

	sessionIdentifier, err := uuid.Parse(sessionID)
	if err != nil {
		logger.Error("invalid session ID format", zap.Error(err))
		return newAPIStatus(http.StatusBadRequest, "invalid session ID format"), StatusFailed
	}

	opts := &logbase.FindSessionOptions{
		ID:             sessionIdentifier,
		UserID:         getUserFromContext(ctx).ID,
		OrganizationID: getOrganizationFromContext(ctx).ID,
	}

	session, err := sh.sessionRepo.List(ctx, opts)
	if err != nil {
		logger.Error("failed to fetch a session by ID", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to list the session by its ID"), StatusFailed
	}

	sessionResponse := &Session{
		ID:             session.ID,
		UserID:         session.UserID,
		UserName:       session.UserName,
		LoginAt:        session.LoginAt,
		OrganizationID: session.OrganizationID,
		LogoutAt:       session.LogoutAt,
		DeviceInfo:     session.DeviceInfo,
		IPAddress:      session.IPAddress,
		Location:       session.Location,
		Status:         logbase.SessionStatus(session.Status),
		CreatedAt:      session.CreatedAt,
	}

	return fetchSessionResponse{
		Session:   sessionResponse,
		APIStatus: newAPIStatus(http.StatusOK, "Session has been fetched successfully"),
	}, StatusSuccess
}

func (sh *sessionHandler) ListAll(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	status := r.URL.Query().Get("status")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	search := r.URL.Query().Get("search")

	opts := &logbase.ListSessionsOptions{
		Paginator: logbase.PaginatorFromRequest(r),
		Status:    status,
		StartDate: startDate,
		EndDate:   endDate,
		Search:    search,
	}

	span.SetAttributes(opts.Paginator.OTELAttributes()...)

	sessions, totalCount, err := sh.sessionRepo.ListAll(ctx, opts)
	if err != nil {
		logger.Error("failed to fetch all sessions", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to fetch all sessions"), StatusFailed
	}

	var sessionResponses []*Session
	for _, session := range sessions {
		sessionResponses = append(sessionResponses, &Session{
			ID:             session.ID,
			UserID:         session.UserID,
			UserName:       session.UserName,
			LoginAt:        session.LoginAt,
			OrganizationID: session.OrganizationID,
			LogoutAt:       session.LogoutAt,
			DeviceInfo:     session.DeviceInfo,
			IPAddress:      session.IPAddress,
			Location:       session.Location,
			Status:         logbase.SessionStatus(session.Status),
			CreatedAt:      session.CreatedAt,
		})
	}

	return fetchAllSessionsResponse{
		Sessions: sessionResponses,
		Meta: meta{
			Paging: pagingInfo{
				Total:   totalCount,
				PerPage: int64(opts.Paginator.PerPage),
				Page:    int64(opts.Paginator.Page),
			},
		},
		APIStatus: newAPIStatus(http.StatusOK, "Sessions have been fetched successfully"),
	}, StatusSuccess
}

func (sh *sessionHandler) Metrics(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("fetching sessions metrics")

	orgID := getOrganizationFromContext(ctx).ID
	opts := logbase.FindSessionOptions{
		OrganizationID: orgID,
	}
	totalCount, suspiciousCount, err := sh.sessionRepo.Metrics(ctx, &opts)
	if err != nil {
		logger.Error("failed to fetch sessions metrics", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to fetch sessions metrics"), StatusFailed
	}

	return sessionMetricsResponse{
		Count:           totalCount,
		SuspiciousCount: suspiciousCount,
		APIStatus:       newAPIStatus(http.StatusOK, "Sessions metrics have been fetched successfully"),
	}, StatusSuccess
}
