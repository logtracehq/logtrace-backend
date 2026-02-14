package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"gitlab.com/logbase/logbase"
	"gitlab.com/logbase/logbase/config"
	queue "gitlab.com/logbase/logbase/internal/pkg/queues"
	"gitlab.com/logbase/logbase/internal/pkg/util"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type eventHander struct {
	cfg       config.Config
	userRepo  logbase.UserRepository
	orgRepo   logbase.OrganizationRepository
	queue     queue.QueueHandler
	eventRepo logbase.EventRepository
}

type createEventRequest struct {
	GenericRequest

	ActionName      string    `json:"action_name"`
	UserID          uuid.UUID `json:"user_id"`
	Username        string    `json:"username"`
	HTTPMethod      string    `json:"http_method"`
	HTTPStatus      string    `json:"http_status"`
	HTTPEndpoint    string    `json:"http_endpoint"`
	ClientIP        string    `json:"client_ip"`
	ClientUserAgent string    `json:"client_user_agent"`
	Type            string    `json:"type"`
	GeoIpLocation   string    `json:"geo_ip_location"`
}

func (e *createEventRequest) Validate() error {
	if util.IsStringEmpty(e.ActionName) {
		return errors.New("action name is required")
	}
	if util.IsStringEmpty(e.HTTPMethod) {
		return errors.New("http method is required")
	}
	if util.IsStringEmpty(e.HTTPStatus) {
		return errors.New("http status is required")
	}
	if util.IsStringEmpty(e.ClientIP) {
		return errors.New("client IP is required")
	}
	if util.IsStringEmpty(e.Username) && util.IsStringEmpty(e.UserID.String()) {
		return errors.New("user information (username or user_id) is required")
	}

	if util.IsStringEmpty(e.ClientUserAgent) {
		return errors.New("client user agent is required")
	}

	// TODO: Add valdiation for the action name types
	return nil
}

func (e *eventHander) Create(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("creating an event")

	req := new(createEventRequest)
	if err := render.Bind(r, req); err != nil {
		return newAPIStatus(http.StatusBadRequest, "failed to validate payload"), StatusFailed
	}

	if err := req.Validate(); err != nil {
		return newAPIStatus(http.StatusBadRequest, err.Error()), StatusFailed
	}

	event := &logbase.Event{
		ActionName:      req.ActionName,
		HTTPMethod:      req.HTTPMethod,
		Username:        req.Username,
		UserID:          req.UserID,
		HTTPStatus:      req.HTTPStatus,
		ClientIP:        req.ClientIP,
		OrganizationID:  getOrganizationFromContext(r.Context()).ID,
		ClientUserAgent: req.ClientUserAgent,
		GeoIPLocation:   req.GeoIpLocation,
		Type:            req.Type,
	}

	err := e.eventRepo.Create(ctx, event)
	if err != nil {
		logger.Error("failed to save the event", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to save the event"), StatusFailed
	}

	return newAPIStatus(http.StatusCreated, "Event has been created successfully"), StatusSuccess
}

func (e *eventHander) List(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("fetch an event by id")

	eventID := chi.URLParam(r, "reference")

	eventIdentifier, err := uuid.Parse(eventID)
	if err != nil {
		logger.Error("invalid event ID format", zap.Error(err))
		return newAPIStatus(http.StatusBadRequest, "invalid event ID format"), StatusFailed
	}

	opts := logbase.ListEventOptions{
		ID:             eventIdentifier,
		OrganizationID: getOrganizationFromContext(r.Context()).ID,
	}

	event, err := e.eventRepo.List(ctx, opts)
	if err != nil {
		logger.Error("failed to fetch event by ID")
		return newAPIStatus(http.StatusInternalServerError, "failed to list the event by it's ID"), StatusFailed
	}

	eventResponse := &Event{
		ID:              event.ID,
		Type:            event.Type,
		Username:        event.Username,
		UserID:          event.UserID.String(),
		HTTPMethod:      event.HTTPMethod,
		HTTPStatus:      event.HTTPStatus,
		HTTPEndpoint:    event.HTTPEndpoint,
		ClientIP:        event.ClientIP,
		OrganizationID:  event.OrganizationID,
		ClientUserAgent: event.ClientUserAgent,
		GeoIPLocation:   event.GeoIPLocation,
		ActionName:      event.ActionName,
		CreatedAt:       event.CreatedAt,
	}

	return fetchEventResponse{
		Event:     eventResponse,
		APIStatus: newAPIStatus(http.StatusOK, "Event has been fetched successfully"),
	}, StatusSuccess
}

func (e *eventHander) ListAll(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	query := r.URL.Query()
	httpStatus := query.Get("http_status")
	httpMethod := query.Get("http_method")
	startDate := query.Get("start_date")
	endDate := query.Get("end_date")
	username := query.Get("username")
	userID := query.Get("user_id")

	parsedUserID, err := uuid.Parse(userID)
	if userID != "" && err != nil {
		logger.Error("invalid user ID format", zap.Error(err))
		return newAPIStatus(http.StatusBadRequest, "invalid user ID format"), StatusFailed
	}

	opts := logbase.ListEventOptions{
		OrganizationID: getOrganizationFromContext(r.Context()).ID,
		Paginator:      logbase.PaginatorFromRequest(r),
		HTTPStatus:     httpStatus,
		HTTPMethod:     httpMethod,
		StartDate:      startDate,
		EndDate:        endDate,
		Username:       username,
		UserID:         parsedUserID,
	}

	events := []*logbase.Event{}

	events, totalCount, err := e.eventRepo.ListAll(ctx, &opts)
	if err != nil {
		return newAPIStatus(http.StatusInternalServerError, "failed to fetch the events"), StatusFailed
	}

	eventResponses := make([]*Event, 0, len(events))
	for _, event := range events {
		eventResponses = append(eventResponses, &Event{
			ID:              event.ID,
			Type:            event.Type,
			HTTPMethod:      event.HTTPMethod,
			HTTPStatus:      event.HTTPStatus,
			Username:        event.Username,
			UserID:          event.UserID.String(),
			HTTPEndpoint:    event.HTTPEndpoint,
			ClientIP:        event.ClientIP,
			OrganizationID:  event.OrganizationID,
			ClientUserAgent: event.ClientUserAgent,
			GeoIPLocation:   event.GeoIPLocation,
			ActionName:      event.ActionName,
			CreatedAt:       event.CreatedAt,
		})
	}

	return fetchEventsResponse{
		Events: eventResponses,
		Meta: meta{
			Paging: pagingInfo{
				Total:   totalCount,
				PerPage: opts.Paginator.PerPage,
				Page:    opts.Paginator.Page,
			},
		},
		APIStatus: newAPIStatus(http.StatusOK, "Events fetched successfully"),
	}, StatusSuccess
}

func (e *eventHander) Metrics(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("fetching events metrics")

	opts := logbase.ListEventOptions{
		OrganizationID: getOrganizationFromContext(r.Context()).ID,
	}

	totalCount, err := e.eventRepo.Metrics(ctx, &opts)
	if err != nil {
		logger.Error("failed to fetch events metrics", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to fetch events metrics"), StatusFailed
	}

	return eventMetricsResponse{
		Count:     totalCount,
		APIStatus: newAPIStatus(http.StatusOK, "Events metrics fetched successfully"),
	}, StatusSuccess
}
