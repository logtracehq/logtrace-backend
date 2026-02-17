package server

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"gitlab.com/logbase/logbase"
	"gitlab.com/logbase/logbase/config"
	"gitlab.com/logbase/logbase/internal/pkg/util"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type planHandler struct {
	cfg      config.Config
	planRepo logbase.PlanRepository
}

type planRequest struct {
	GenericRequest

	Name        string   `json:"name"`
	Price       float64  `json:"price"`
	Description string   `json:"description"`
	Features    []string `json:"features"`
	Period      string   `json:"period"`
	CTA         string   `json:"cta"`
}

func (p *planRequest) Validate() error {
	if util.IsStringEmpty(p.Name) {
		return errors.New("plan name is required")
	}

	return nil
}

func (p *planHandler) Create(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("creating an plan")

	req := new(planRequest)
	if err := render.Bind(r, req); err != nil {
		return newAPIStatus(http.StatusBadRequest, "failed to validate payload"), StatusFailed
	}

	if err := req.Validate(); err != nil {
		return newAPIStatus(http.StatusBadRequest, err.Error()), StatusFailed
	}

	features := make([]logbase.Feature, len(req.Features))
	for i, f := range req.Features {
		features[i] = logbase.Feature(strings.ToLower(f))
	}

	plan := &logbase.Plan{
		Name:        req.Name,
		Price:       req.Price,
		Description: req.Description,
		Features:    features,
		CTA:         req.CTA,
		Period:      req.Period,
	}

	err := p.planRepo.Create(ctx, plan)
	if err != nil {
		logger.Error("failed to save the plan", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to save the plan"), StatusFailed
	}

	return newAPIStatus(http.StatusCreated, "plan has been created successfully"), StatusSuccess
}

func (p *planHandler) Get(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("fetch a plan by id")

	ref := chi.URLParam(r, "reference")

	planId, err := uuid.Parse(ref)
	if err != nil {
		logger.Error("invalid plan ID format", zap.Error(err))
		return newAPIStatus(http.StatusBadRequest, "invalid plan ID format"), StatusFailed
	}
	plan, err := p.planRepo.List(ctx, planId)
	if err != nil {
		logger.Error("failed to fetch plan by ID")
		return newAPIStatus(http.StatusInternalServerError, "failed to list the plan by it's ID"), StatusFailed
	}

	return fetchPlanResponse{
		Plan:      plan,
		APIStatus: newAPIStatus(http.StatusOK, "Plan has been fetched successfully"),
	}, StatusSuccess
}

func (p *planHandler) ListAll(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("fetch all plans")

	plans, err := p.planRepo.ListAll(ctx)
	if err != nil {
		logger.Error("failed to fetch all plans")
		return newAPIStatus(http.StatusInternalServerError, "failed to list all plans"), StatusFailed
	}

	return fetchPlansResponse{
		Plans:     plans,
		APIStatus: newAPIStatus(http.StatusOK, "Plans have been fetched successfully"),
	}, StatusSuccess
}

func (p *planHandler) Update(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("updating a plan")

	ref := chi.URLParam(r, "reference")

	planId, err := uuid.Parse(ref)
	if err != nil {
		logger.Error("invalid plan ID format", zap.Error(err))
		return newAPIStatus(http.StatusBadRequest, "invalid plan ID format"), StatusFailed
	}

	req := new(planRequest)
	if err := render.Bind(r, req); err != nil {
		return newAPIStatus(http.StatusBadRequest, "failed to validate payload"), StatusFailed
	}

	if err := req.Validate(); err != nil {
		return newAPIStatus(http.StatusBadRequest, err.Error()), StatusFailed
	}

	features := make([]logbase.Feature, len(req.Features))
	for i, f := range req.Features {
		features[i] = logbase.Feature(strings.ToLower(f))
	}

	plan := &logbase.Plan{
		ID:          planId,
		Name:        req.Name,
		Price:       req.Price,
		Description: req.Description,
		Features:    features,
		CTA:         req.CTA,
		Period:      req.Period,
	}

	updatedPlan, err := p.planRepo.Update(ctx, plan)
	if err != nil {
		logger.Error("failed to update the plan", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to update the plan"), StatusFailed
	}

	return fetchPlanResponse{
		Plan:      updatedPlan,
		APIStatus: newAPIStatus(http.StatusOK, "Plan has been updated successfully"),
	}, StatusSuccess
}

func (p *planHandler) Delete(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("deleting a plan")

	ref := chi.URLParam(r, "reference")

	planId, err := uuid.Parse(ref)
	if err != nil {
		logger.Error("invalid plan ID format", zap.Error(err))
		return newAPIStatus(http.StatusBadRequest, "invalid plan ID format"), StatusFailed
	}

	err = p.planRepo.Delete(ctx, planId)
	if err != nil {
		logger.Error("failed to delete the plan", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to delete the plan"), StatusFailed
	}

	return newAPIStatus(http.StatusOK, "Plan has been deleted successfully"), StatusSuccess
}
