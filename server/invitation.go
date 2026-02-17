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

type invitationHandler struct {
	cfg            config.Config
	invitationRepo logbase.InvitationRepository
	userRepo       logbase.UserRepository
}

type createInvitationRequest struct {
	GenericRequest

	Fullname string `json:"fullname"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

func (c *createInvitationRequest) Validate() error {
	if util.IsStringEmpty(c.Fullname) {
		return errors.New("fullname is required")
	}

	if util.IsStringEmpty(c.Email) {
		return errors.New("email is required")
	}
	if util.IsStringEmpty(c.Role) {
		return errors.New("role is required")
	}

	return nil
}

func (i *invitationHandler) Create(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("creating an invitation")

	req := new(createInvitationRequest)
	if err := render.Bind(r, req); err != nil {
		return newAPIStatus(http.StatusBadRequest, "failed to validate payload"), StatusFailed
	}

	if err := req.Validate(); err != nil {
		return newAPIStatus(http.StatusBadRequest, err.Error()), StatusFailed
	}

	orgID := getOrganizationFromContext(ctx).ID

	invitation := &logbase.Invitation{
		Fullname:       req.Fullname,
		Email:          logbase.Email(req.Email),
		OrganizationID: orgID,
		Role:           logbase.RoleName(req.Role),
		Status:         "PENDING",
		Token:          "token", // TODO: generate a secure token
	}

	err := i.invitationRepo.Create(ctx, invitation)
	if err != nil {
		logger.Error("failed to create invitation", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to create invitation"), StatusFailed
	}

	return newAPIStatus(http.StatusCreated, "Invitation created successfully"), StatusSuccess
}

func (i *invitationHandler) List(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("listing invitations")

	opts := logbase.ListInvitationOptions{
		OrganizationID: getOrganizationFromContext(ctx).ID,
	}

	invitations, err := i.invitationRepo.List(ctx, opts)
	if err != nil {
		logger.Error("failed to list invitations", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to list invitations"), StatusFailed
	}

	return &fetchInvitationsResponse{
		Invitations: invitations,
		APIStatus:   newAPIStatus(http.StatusOK, "Invitations fetched successfully"),
	}, StatusSuccess
}
