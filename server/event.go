package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/terra-consults/logbase"
	"github.com/terra-consults/logbase/config"
	queue "github.com/terra-consults/logbase/internal/pkg/queues"
	"github.com/terra-consults/logbase/internal/pkg/util"
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

	ActionName      string
	Username        string
	HTTPMethod      string
	HTTPStatus      string
	HTTPEndpoint    string
	ClientIP        string
	ClientUserAgent string
	Type            string
	GeoIpLocation   string
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

	event := logbase.Event{
		ActionName:      req.ActionName,
		HTTPMethod:      req.HTTPMethod,
		HTTPStatus:      req.HTTPStatus,
		ClientIP:        req.ClientIP,
		ClientUserAgent: req.ClientUserAgent,
		GeoIPLocation:   req.GeoIpLocation,
		Type:            req.Type,
	}

	err := e.eventRepo.Create(ctx, event)
	if err != nil {
		logger.Error("failed to save the event", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to save the event"), StatusFailed
	}

	return newAPIStatus(http.StatusCreated, "event has been created successfully"), StatusSuccess
}

func (e *eventHander) ListByID(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("fetch an event by id")

	eventID := chi.URLParam(r, "reference")

	eventIdentifier, _ := uuid.Parse(eventID)

	opts := logbase.EventFindOptions{
		ID: eventIdentifier,
	}

	event, err := e.eventRepo.List(ctx, opts)
	if err != nil {
		logger.Error("failed to fetch event by ID")
		return newAPIStatus(http.StatusInternalServerError, "failed to list the event by it's ID"), StatusFailed
	}

	return fetchEventResponse{
		Event:     event,
		APIStatus: newAPIStatus(http.StatusOK, "Event has been fetched successfully"),
	}, StatusSuccess
}

func (e *eventHander) List(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	events, err := e.eventRepo.ListAll(ctx, 10, 10)
	if err != nil {
		return newAPIStatus(http.StatusInternalServerError, "failed to fetch the events"), StatusFailed
	}

	return fetchEventsResponse{
		Events:    events,
		APIStatus: newAPIStatus(http.StatusOK, "Fetch the events successfully"),
	}, StatusSuccess
}
