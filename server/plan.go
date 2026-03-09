package server

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"gitlab.com/logtrace/logtrace"
	"gitlab.com/logtrace/logtrace/config"
	"gitlab.com/logtrace/logtrace/internal/pkg/util"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type planHandler struct {
	cfg      config.Config
	planRepo logtrace.PlanRepository
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

// @Description Create a new plan
// @Tags Plans
// @Accept json
// @Produce json
// @Param plan body planRequest true "Plan creation request"
// @Success 201 {object} APIStatus "Plan created successfully"
// @Failure 400 {object} APIStatus "Invalid request body"
// @Failure 500 {object} APIStatus "Could not create plan at this time. an error occurred"
// @Router /v1/plans [post]
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

	features := make([]logtrace.Feature, len(req.Features))
	for i, f := range req.Features {
		features[i] = logtrace.Feature(strings.ToLower(f))
	}

	plan := &logtrace.Plan{
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

// @Description Get a specific plan by ID
// @Tags Plans
// @Accept json
// @Produce json
// @Param reference path string true "Plan ID"
// @Success 200 {object} fetchPlanResponse "Plan has been fetched successfully"
// @Failure 400 {object} APIStatus "Invalid plan reference"
// @Failure 404 {object} APIStatus "Plan not found"
// @Failure 500 {object} APIStatus "Failed to fetch plan"
// @Router /v1/plans/{reference} [get]
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
		logger.Error("failed to fetch plan by ID", zap.Error(err))
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
		logger.Error("failed to fetch all plans", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to list all plans"), StatusFailed
	}

	return fetchPlansResponse{
		Plans:     plans,
		APIStatus: newAPIStatus(http.StatusOK, "Plans have been fetched successfully"),
	}, StatusSuccess
}

// @Description Update an existing plan by ID
// @Tags Plans
// @Accept json
// @Produce json
// @Param reference path string true "Plan ID"
// @Param plan body planRequest true "Plan update request"
// @Success 200 {object} fetchPlanResponse "Plan has been updated successfully"
// @Failure 400 {object} APIStatus "Invalid plan reference or request body"
// @Failure 404 {object} APIStatus "Plan not found"
// @Failure 500 {object} APIStatus "Failed to update plan"
// @Router /v1/plans/{reference} [put]
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

	features := make([]logtrace.Feature, len(req.Features))
	for i, f := range req.Features {
		features[i] = logtrace.Feature(strings.ToLower(f))
	}

	plan := &logtrace.Plan{
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

// @Description Delete an existing plan by ID
// @Tags Plans
// @Accept json
// @Produce json
// @Param reference path string true "Plan ID"
// @Success 200 {object} APIStatus "Plan has been deleted successfully"
// @Failure 400 {object} APIStatus "Invalid plan reference"
// @Failure 404 {object} APIStatus "Plan not found"
// @Failure 500 {object} APIStatus "Failed to delete plan"
// @Router /v1/plans/{reference} [delete]
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
