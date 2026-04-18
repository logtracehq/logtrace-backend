package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"gitlab.com/logtrace/logtrace"
	"gitlab.com/logtrace/logtrace/config"
	"gitlab.com/logtrace/logtrace/internal/pkg/util"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type invitationHandler struct {
	cfg            config.Config
	invitationRepo logtrace.InvitationRepository
	userRepo       logtrace.UserRepository
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

// @Description Create a new invitation
// @Tags Invitations
// @Accept json
// @Produce json
// @Param invitation body createInvitationRequest true "Invitation creation request"
// @Success 201 {object} APIStatus "Invitation created successfully"
// @Failure 400 {object} APIStatus "Invalid request body"
// @Failure 500 {object} APIStatus "Could not create invitation at this time. an error occurred"
// @Router /v1/invitations [post]
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

	invitation := &logtrace.Invitation{
		Fullname:       req.Fullname,
		Email:          logtrace.Email(req.Email),
		OrganizationID: orgID,
		Role:           logtrace.RoleName(req.Role),
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

// @Description Accept an invitation by token
// @Tags Invitations
// @Accept json
// @Produce json
// @Param token path string true "Invitation token"
// @Success 200 {object} APIStatus "Invitation accepted successfully"
// @Failure 400 {object} APIStatus "Invalid invitation token"
// @Failure 404 {object} APIStatus "Invitation not found"
// @Failure 500 {object} APIStatus "Failed to accept invitation"
// @Router /v1/invitations/{token}/accept [post]
func (i *invitationHandler) List(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("listing invitations")

	opts := logtrace.ListInvitationOptions{
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

// @Description Delete an invitation by ID
// @Tags Invitations
// @Produce json
// @Param reference path string true "Invitation reference"
// @Success 200 {object} APIStatus "Invitation deleted successfully"
// @Failure 400 {object} APIStatus "Invalid invitation reference"
// @Failure 404 {object} APIStatus "Invitation not found"
// @Failure 500 {object} APIStatus "Failed to delete invitation"
// @Router /v1/invitations/{reference} [delete]
func (i *invitationHandler) Delete(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("deleting invitation")

	ref := chi.URLParam(r, "reference")
	invitationID, err := uuid.Parse(ref)
	if err != nil {
		return newAPIStatus(http.StatusBadRequest, "invalid invitation reference"), StatusFailed
	}

	orgID := getOrganizationFromContext(ctx).ID

	opts := logtrace.FindInvitationOptions{
		ID:             invitationID,
		OrganizationID: orgID,
	}

	if err := i.invitationRepo.Delete(ctx, opts); err != nil {
		logger.Error("failed to delete invitation", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to delete invitation"), StatusFailed
	}

	return newAPIStatus(http.StatusOK, "Invitation deleted successfully"), StatusSuccess
}
