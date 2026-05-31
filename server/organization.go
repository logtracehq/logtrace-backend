package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"gitlab.com/logtrace/logtrace"
	"gitlab.com/logtrace/logtrace/config"
	"gitlab.com/logtrace/logtrace/internal/pkg/util"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type orgHandler struct {
	cfg      config.Config
	orgRepo  logtrace.OrganizationRepository
	userRepo logtrace.UserRepository
}

type createOrgRequest struct {
	GenericRequest

	Name     string `json:"name"`
	ImageURL string `json:"image_url"`
}

type updateOrgRequest struct {
	GenericRequest

	Name                  string `json:"name"`
	ImageURL              string `json:"image_url"`
	IsSubscriptionActive  bool   `json:"is_subscription_active"`
	PlanName              string `json:"plan_name"`
	IsActive              bool   `json:"is_active"`
	SubscriptionExpiresAt time.Time
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

	twoWeeksLater := time.Now().AddDate(0, 0, 14)
	org := &logtrace.Organization{
		Name:                  req.Name,
		IsActive:              true,
		IsSubscriptionActive:  false,
		PlanName:              "Free",
		ImageURL:              req.ImageURL,
		SubscriptionExpiresAt: &twoWeeksLater,
	}

	org, err := o.orgRepo.Create(ctx, org)
	if err != nil {
		logger.Error("an error occurred while creating organization", zap.Error(err))
		return newAPIStatus(
			http.StatusInternalServerError,
			"Could not create organization at this time. an error occurred",
		), StatusFailed
	}

	user := getUserFromContext(ctx)
	if user.Metadata == nil {
		user.Metadata = &logtrace.UserMetadata{}
	}
	user.Metadata.OrganizationID = append(user.Metadata.OrganizationID, org.ID)
	if _, err := o.userRepo.Update(ctx, user); err != nil {
		logger.Error("an error occurred while updating user metadata after org creation", zap.Error(err))
		return newAPIStatus(
			http.StatusInternalServerError,
			"Organization created but could not update user membership",
		), StatusFailed
	}

	return newAPIStatus(http.StatusCreated, "Organization created successfully"), StatusSuccess
}

// @Description Update an existing organization by ID
// @Tags Organization
// @Accept json
// @Produce json
// @Param reference path string true "Organization ID"
// @Param organization body updateOrgRequest true "Organization update request"
// @Success 200 {object} OrganizationResponse "Organization updated successfully"
// @Failure 400 {object} APIStatus "Invalid organization reference or request body"
// @Failure 404 {object} APIStatus "Organization not found"
// @Failure 500 {object} APIStatus "Failed to update organization"
// @Router /v1/organizations/{reference} [put]
func (o *orgHandler) updateOrganization(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("updating an organization")

	organizationID := getOrganizationFromContext(ctx).ID
	req := new(updateOrgRequest)

	opts := logtrace.FindOrganizationOptions{
		ID: organizationID,
	}

	if err := render.Bind(r, req); err != nil {
		return newAPIStatus(http.StatusBadRequest, "invalid request body"), StatusFailed
	}

	org := &logtrace.Organization{
		ID:                    opts.ID,
		Name:                  req.Name,
		ImageURL:              req.ImageURL,
		IsSubscriptionActive:  req.IsSubscriptionActive,
		SubscriptionExpiresAt: &req.SubscriptionExpiresAt,
		PlanName:              req.PlanName,
		IsActive:              req.IsActive,
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

// @Description Delete an existing organization by ID
// @Tags Organization
// @Accept json
// @Produce json
// @Param reference path string true "Organization ID"
// @Success 200 {object} APIStatus "Organization deleted successfully"
// @Failure 400 {object} APIStatus "Invalid organization reference"
// @Failure 404 {object} APIStatus "Organization not found"
// @Failure 500 {object} APIStatus "Failed to delete organization"
// @Router /v1/organizations/{reference} [delete]
func (o *orgHandler) deleteOrganization(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("deleting an organization")

	organizationID := getOrganizationFromContext(ctx).ID

	opts := logtrace.FindOrganizationOptions{
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
