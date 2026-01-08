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
	"gitlab.com/logbase/logbase/internal/pkg/util"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type resourceHandler struct {
	cfg          config.Config
	userRepo     logbase.UserRepository
	resourceRepo logbase.ResourceRepository
}

type createResourceRequest struct {
	GenericRequest

	Name string
	Type string
}

func (r *createResourceRequest) Validate() error {
	if util.IsStringEmpty(r.Name) {
		return errors.New("resource name is required")
	}
	if util.IsStringEmpty(r.Type) {
		return errors.New("resource type is required")
	}

	return nil
}

func (res *resourceHandler) Create(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("creating new resource")

	req := new(createResourceRequest)
	if err := render.Bind(r, req); err != nil {
		logger.Error("failed to bind create resource request", zap.Error(err))
		return newAPIStatus(http.StatusBadRequest, "failed to create resource request"), StatusFailed
	}

	if err := req.Validate(); err != nil {
		logger.Error("invalid create resource request", zap.Error(err))
		return newAPIStatus(http.StatusBadRequest, "failed to validate payload"), StatusFailed
	}

	resource := &logbase.Resource{
		Name: req.Name,
		Type: req.Type,
	}

	if err := res.resourceRepo.Create(ctx, resource); err != nil {
		logger.Error("failed to create resource", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to create resource"), StatusFailed
	}

	return newAPIStatus(http.StatusCreated, "Resource created successfully"), StatusSuccess
}

func (res *resourceHandler) List(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	resourceID := chi.URLParam(r, "reference")
	resourceIdentifier, err := uuid.Parse(resourceID)
	if err != nil {
		logger.Error("invalid resource id", zap.Error(err))
		return newAPIStatus(http.StatusBadRequest, "invalid resource reference"), StatusFailed
	}

	resource, err := res.resourceRepo.Get(ctx, resourceIdentifier)
	if err != nil {
		logger.Error("failed to fetch resource", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to fetch resource"), StatusFailed
	}

	return fetchResource{
		Resource:  *resource,
		APIStatus: newAPIStatus(http.StatusOK, "resource fetched successfully"),
	}, StatusSuccess
}

func (res *resourceHandler) Delete(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	resourceID := r.URL.Query().Get("reference")
	resourceIdentifier, err := uuid.Parse(resourceID)
	if err != nil {
		logger.Error("invalid resource id", zap.Error(err))
		return newAPIStatus(http.StatusBadRequest, "invalid resource id"), StatusFailed
	}

	if err := res.resourceRepo.Delete(ctx, resourceIdentifier); err != nil {
		logger.Error("failed to delete resource", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to delete resource"), StatusFailed
	}

	return newAPIStatus(http.StatusOK, "Resource has been deleted successfully"), StatusSuccess
}

func (res *resourceHandler) ListAll(ctx context.Context, span trace.Span, logger *zap.Logger, w http.ResponseWriter,
	r *http.Request,
) (render.Renderer, Status) {
	opts := logbase.ListResourceOptions{
		Paginator: logbase.PaginatorFromRequest(r),
	}

	span.SetAttributes(opts.Paginator.OTELAttributes()...)

	resources, totalCount, err := res.resourceRepo.ListAll(ctx, opts)
	if err != nil {
		logger.Error("failed to fetch all resources", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to fetch all resources"), StatusFailed
	}

	return fetchAllResources{
		Resources: resources,
		Meta: meta{
			Paging: pagingInfo{
				Total:   totalCount,
				Page:    int64(opts.Paginator.Page),
				PerPage: int64(opts.Paginator.PerPage),
			},
		},
		APIStatus: newAPIStatus(http.StatusOK, "Resources fetched successfully"),
	}, StatusSuccess
}
