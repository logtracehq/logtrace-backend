package server

import (
	"context"
	"net/http"

	"github.com/go-chi/render"
	"gitlab.com/logbase/logbase/config"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Status uint8

const (
	StatusSuccess Status = iota
	StatusFailed
)

// logbaseHTTPHandler is a wrapper for HTTP handlers
// that helps to centralizes error handling, otel tracing amongst others
type LogbaseHTTPHandler func(
	context.Context,
	trace.Span,
	*zap.Logger,
	http.ResponseWriter,
	*http.Request) (render.Renderer, Status)

// WrapLogbaseHTTPHandler is a middleware that wraps our handlers and manages errors
func WrapLogbaseHTTPHandler(logger *zap.Logger, handler LogbaseHTTPHandler, cfg config.Config, spanName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span, rid := getTracer(r.Context(), r, spanName)
		defer span.End()

		logger := logger.With(zap.String("request_id", rid))

		if doesOrganizationExistInContext(r.Context()) {
			organization := getOrganizationFromContext(r.Context()).ID.String()
			logger = logger.With(zap.String("organization_id", organization))
			span.SetAttributes(attribute.String("organization_id", organization))
		}

		if doesUserExistInContext(r.Context()) {
			userID := getUserFromContext(r.Context()).ID.String()
			logger = logger.With(zap.String("user_id", userID))
			span.SetAttributes(
				attribute.String("user_id", userID))
		}

		resp, status := handler(ctx, span, logger, w, r)
		switch status {
		case StatusFailed:
			span.SetStatus(codes.Error, "")
		case StatusSuccess:
			span.SetStatus(codes.Ok, "")
		default:
			_ = render.Render(w, r, newAPIStatus(http.StatusInternalServerError, "unknown error"))
			return
		}

		err := render.Render(w, r, resp)
		if err != nil {
			logger.Error("could not write http response", zap.Error(err))
		}
	}
}
