package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/render"
	"gitlab.com/logbase/logbase"
	"gitlab.com/logbase/logbase/config"
	"gitlab.com/logbase/logbase/internal/pkg/util"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type orgHandler struct {
	cfg      config.Config
	orgRepo  logbase.OrganizationRepository
	userRepo logbase.UserRepository
}

type createOrgRequest struct {
	GenericRequest

	Name     string `json:"name"`
	ImageURL string `json:"image_url"`
}

type updateOrgRequest struct {
	GenericRequest

	Name                 string `json:"name"`
	ImageURL             string `json:"image_url"`
	IsSubscriptionActive bool   `json:"is_subscription_active"`
	PlanName             string `json:"plan_name"`
	IsActive             bool   `json:"is_active"`
}

func (c *createOrgRequest) Validate() error {
	if util.IsStringEmpty(c.Name) {
		return errors.New("organization name is required")
	}
	return nil
}

// @Description Create a new organization
// @Tags Organization
// @Accept json
// @Produce json
// @Param organization body createOrgRequest true "Organization creation request"
// @Success 201 {object} APIStatus "Organization created successfully"
// @Failure 400 {object} APIStatus "Invalid request body"
// @Failure 500 {object} APIStatus "Could not create organization at this time. an error occurred"
// @Router /v1/organizations [post]
func (o *orgHandler) createOrganization(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("creating an organization")

	req := new(createOrgRequest)

	if err := render.Bind(r, req); err != nil {
		return newAPIStatus(http.StatusBadRequest, "invalid request body"), StatusFailed
	}

	if err := req.Validate(); err != nil {
		return newAPIStatus(http.StatusBadRequest, err.Error()), StatusFailed
	}

	org := &logbase.Organization{
		Name:                 req.Name,
		IsActive:             true,
		IsSubscriptionActive: false,
		PlanName:             "free",
		ImageURL:             req.ImageURL,
	}

	org, err := o.orgRepo.Create(ctx, org)
	if err != nil {
		logger.Error("an error occurred while creating organization", zap.Error(err))
		return newAPIStatus(
			http.StatusInternalServerError,
			"Could not create organization at this time. an error occurred",
		), StatusFailed
	}

	return newAPIStatus(http.StatusCreated, "Organization created successfully"), StatusSuccess
}

func (o *orgHandler) updateOrganization(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("updating an organization")

	organizationID := getOrganizationFromContext(ctx).ID
	req := new(updateOrgRequest)

	opts := logbase.FindOrganizationOptions{
		ID: organizationID,
	}

	if err := render.Bind(r, req); err != nil {
		return newAPIStatus(http.StatusBadRequest, "invalid request body"), StatusFailed
	}

	org := &logbase.Organization{
		ID:                   opts.ID,
		Name:                 req.Name,
		ImageURL:             req.ImageURL,
		IsSubscriptionActive: req.IsSubscriptionActive,
		PlanName:             req.PlanName,
		IsActive:             req.IsActive,
	}

	updatedOrg, err := o.orgRepo.Update(ctx, org)
	if err != nil {
		logger.Error("an error occurred while updating organization", zap.Error(err))
		return newAPIStatus(
			http.StatusInternalServerError,
			"Could not update organization at this time. an error occurred",
		), StatusFailed
	}

	return OrganizationResponse{
		Organization: updatedOrg,
		APIStatus:    newAPIStatus(http.StatusOK, "Organization updated successfully"),
	}, StatusSuccess
}

func (o *orgHandler) deleteOrganization(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("deleting an organization")

	organizationID := getOrganizationFromContext(ctx).ID

	opts := logbase.FindOrganizationOptions{
		ID: organizationID,
	}

	err := o.orgRepo.Delete(ctx, &opts)
	if err != nil {
		logger.Error("an error occurred while deleting organization", zap.Error(err))
		return newAPIStatus(
			http.StatusInternalServerError,
			"Could not delete organization at this time. an error occurred",
		), StatusFailed
	}

	return newAPIStatus(http.StatusOK, "Organization deleted successfully"), StatusSuccess
}
