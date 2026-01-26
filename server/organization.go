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

	Name string `json:"name"`
}

func (c *createOrgRequest) Validate() error {
	if util.IsStringEmpty(c.Name) {
		return errors.New("organization name is required")
	}
	return nil
}

func (a *authHandler) createOrganization(ctx context.Context, span trace.Span, logger *zap.Logger,
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
	}

	org, err := a.orgRepo.Create(ctx, org)
	if err != nil {
		logger.Error("an error occurred while creating organization", zap.Error(err))
		return newAPIStatus(
			http.StatusInternalServerError,
			"Could not create organization at this time. an error occurred",
		), StatusFailed
	}

	return newAPIStatus(http.StatusCreated, "Organization created successfully"), StatusSuccess
}
