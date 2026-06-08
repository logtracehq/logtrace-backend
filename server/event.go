package server

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"gitlab.com/logtrace/logtrace"
	"gitlab.com/logtrace/logtrace/config"
	queue "gitlab.com/logtrace/logtrace/internal/pkg/queues"
	"gitlab.com/logtrace/logtrace/internal/pkg/services"
	"gitlab.com/logtrace/logtrace/internal/pkg/util"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type eventHander struct {
	cfg         config.Config
	userRepo    logtrace.UserRepository
	orgRepo     logtrace.OrganizationRepository
	queue       queue.QueueHandler
	eventRepo   logtrace.EventRepository
	orgUserRepo logtrace.OrganizationUserRepository
}

type createEventRequest struct {
	GenericRequest

	Name           string                  `json:"name"`
	UserID         string                  `json:"user_id"`
	Username       string                  `json:"username"`
	RequestDetails logtrace.RequestDetails `json:"request_details"`
	Type           string                  `json:"type"`
	Metadata       logtrace.Metadata       `json:"metadata"`
}

func (e *createEventRequest) Validate() error {
	if util.IsStringEmpty(e.Name) {
		return errors.New("event name is required")
	}

	if util.IsStringEmpty(e.Username) && util.IsStringEmpty(e.UserID) {
		return errors.New("user information (username or user_id) is required")
	}

	return nil
}

// @Description Create a new event log entry
// @Tags Events
// @Accept json
// @Produce json
// @Param event body createEventRequest true "Event creation request"
// @Success 201 {object} APIStatus "Event created successfully"
// @Failure 400 {object} APIStatus "Invalid request body"
// @Failure 500 {object} APIStatus "Could not create event at this time. an error occurred"
// @Router /v1/events [post]
func (e *eventHander) Create(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("creating an event")

	req := new(createEventRequest)
	if err := render.Bind(r, req); err != nil {
		return newAPIStatus(http.StatusBadRequest, "failed to validate payload"), StatusFailed
	}

	if err := req.Validate(); err != nil {
		return newAPIStatus(http.StatusBadRequest, "failed to validate request payload: %v", err.Error()), StatusFailed
	}

	geo, err := services.GeoIPLookup(ctx, req.RequestDetails.IPAddress)
	if err != nil {
		logger.Warn("failed to resolve geoip", zap.Error(err))
	}

	geoLocation := ""
	if geo != nil {
		geoLocation = geo.Country
		if geo.City != "" {
			geoLocation += ", " + geo.City
		}
	}

	req.RequestDetails.GeoIPLocation = geoLocation

	event := &logtrace.Event{
		Name:           req.Name,
		Username:       req.Username,
		UserID:         req.UserID,
		OrganizationID: getOrganizationFromContext(r.Context()).ID,
		Type:           req.Type,
		Metadata:       req.Metadata,
		RequestDetails: req.RequestDetails,
	}

	orgUser, err := e.orgUserRepo.Find(ctx, &logtrace.FindOrganizationUserOptions{
		UserID:         event.UserID,
		OrganizationID: event.OrganizationID,
		Username:       event.Username,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logger.Error("failed to find organization user", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to find organization user"), StatusFailed
	}

	if orgUser != nil {
		event.UserID = orgUser.UserID
		event.Username = orgUser.Username
	} else {
		_, err := e.orgUserRepo.Create(ctx, &logtrace.OrganizationUser{
			UserID:         event.UserID,
			OrganizationID: event.OrganizationID,
			Username:       event.Username,
		})
		if err != nil {
			logger.Error("failed to create organization user", zap.Error(err))
			return newAPIStatus(http.StatusInternalServerError, "failed to create organization user"), StatusFailed
		}
	}

	event.UserID = req.UserID
	event.Username = req.Username

	if err := e.queue.Add(ctx, queue.QueueTopicSaveEvent, queue.SaveEventOptions{Event: event}); err != nil {
		logger.Error("failed to enqueue event", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to enqueue event"), StatusFailed
	}

	return newAPIStatus(http.StatusAccepted, "Event accepted for processing"), StatusSuccess
}

// @Description Get a specific event by ID
// @Tags Events
// @Accept json
// @Produce json
// @Param reference path string true "Event ID"
// @Success 200 {object} fetchEventResponse "Event has been fetched successfully"
// @Failure 400 {object} APIStatus "Invalid event reference"
// @Failure 404 {object} APIStatus "Event not found"
// @Failure 500 {object} APIStatus "Failed to fetch event"
// @Router /v1/events/{reference} [get]
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

	opts := logtrace.ListEventOptions{
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
		UserID:          event.UserID,
		HTTPMethod:      event.RequestDetails.HTTPMethod,
		HTTPStatusCode:  event.RequestDetails.HTTPStatusCode,
		HTTPEndpoint:    event.RequestDetails.HTTPEndpoint,
		IPAddress:       event.RequestDetails.IPAddress,
		ClientUserAgent: event.RequestDetails.ClientUserAgent,
		GeoIPLocation:   event.RequestDetails.GeoIPLocation,
		Name:            event.Name,
		Metadata:        event.Metadata,
		CreatedAt:       event.CreatedAt,
		OperatingSystem: event.RequestDetails.OperatingSystem,
	}

	return fetchEventResponse{
		Event:     eventResponse,
		APIStatus: newAPIStatus(http.StatusOK, "Event has been fetched successfully"),
	}, StatusSuccess
}

// @Description Get a list of events with optional filters and pagination
// @Tags Events
// @Accept json
// @Produce json
// @Param http_status query string false "Filter by HTTP status"
// @Param http_method query string false "Filter by HTTP method"
// @Param start_date query string false "Filter by start date (ISO format)"
// @Param end_date query string false "Filter by end date (ISO format)"
// @Param username query string false "Filter by username"
// @Param user_id query string false "Filter by user ID"
// @Param search query string false "Search term for action name or endpoint"
// @Success 200 {object} fetchEventsResponse "Events fetched successfully"
// @Failure 500 {object} APIStatus "Failed to fetch events"
// @Router /v1/events [get]
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
	search := query.Get("search")

	opts := logtrace.ListEventOptions{
		OrganizationID: getOrganizationFromContext(r.Context()).ID,
		Paginator:      logtrace.PaginatorFromRequest(r),
		HTTPStatus:     httpStatus,
		HTTPMethod:     httpMethod,
		StartDate:      startDate,
		EndDate:        endDate,
		Username:       username,
		UserID:         userID,
		Search:         search,
	}

	events := []*logtrace.Event{}

	events, totalCount, err := e.eventRepo.ListAll(ctx, &opts)
	if err != nil {
		return newAPIStatus(http.StatusInternalServerError, "failed to fetch the events"), StatusFailed
	}

	eventResponses := make([]*Event, 0, len(events))
	for _, event := range events {
		eventResponses = append(eventResponses, &Event{
			ID:              event.ID,
			Type:            event.Type,
			HTTPMethod:      event.RequestDetails.HTTPMethod,
			HTTPStatusCode:  event.RequestDetails.HTTPStatusCode,
			Username:        event.Username,
			UserID:          event.UserID,
			HTTPEndpoint:    event.RequestDetails.HTTPEndpoint,
			IPAddress:       event.RequestDetails.IPAddress,
			ClientUserAgent: event.RequestDetails.ClientUserAgent,
			GeoIPLocation:   event.RequestDetails.GeoIPLocation,
			Name:            event.Name,
			Metadata:        event.Metadata,
			CreatedAt:       event.CreatedAt,
			OperatingSystem: event.RequestDetails.OperatingSystem,
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

// @Description Get metrics for events
// @Tags Events
// @Accept json
// @Produce json
// @Success 200 {object} eventMetricsResponse "Events metrics fetched successfully"
// @Failure 500 {object} APIStatus "Failed to fetch events metrics"
// @Router /v1/events/metrics [get]
func (e *eventHander) Metrics(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("fetching events metrics")

	opts := logtrace.ListEventOptions{
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

func (e *eventHander) TopActorMetrics(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	// @Description Get top actor metrics for events in the last 24 hours
	// @Tags Events
	// @Accept json
	// @Produce json
	// @Success 200 {object} eventTopActorMetricsResponse "Top actor metrics fetched successfully"
	// @Failure 500 {object} APIStatus "Failed to fetch top actor metrics"
	// @Router /v1/metrics/events/top-actors [get]
	logger.Debug("fetching top actor metrics for events")

	opts := logtrace.ListEventOptions{
		OrganizationID: getOrganizationFromContext(r.Context()).ID,
	}

	actors, err := e.eventRepo.TopActorMetrics(ctx, &opts)
	if err != nil {
		logger.Error("failed to fetch top actor metrics", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to fetch top actor metrics"), StatusFailed
	}

	return eventTopActorMetricsResponse{
		Actors:    actors,
		APIStatus: newAPIStatus(http.StatusOK, "Top actor metrics fetched successfully"),
	}, StatusSuccess
}
