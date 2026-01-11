package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/microcosm-cc/bluemonday"
	"gitlab.com/logbase/logbase"
	"gitlab.com/logbase/logbase/config"
	"gitlab.com/logbase/logbase/internal/pkg/util"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type apiKeyHandler struct {
	apiKeyRepo logbase.APIKeyRepository
	cfg        config.Config
}

type createAPIKeyRequest struct {
	GenericRequest

	Name string `json:"name,omitempty" validate:"required"`
}

func (c *createAPIKeyRequest) Validate() error {
	if util.IsStringEmpty(c.Name) {
		return errors.New("please provide the name of the api key")
	}

	if len(c.Name) < 3 {
		return errors.New("name must be more than 3 characters")
	}

	if len(c.Name) > 20 {
		return errors.New("name must be more than 20 characters")
	}

	p := bluemonday.StrictPolicy()

	c.Name = p.Sanitize(c.Name)

	return nil
}

// @Description Creates a new api key
// @Tags developers
// @Accept  json
// @Produce  json
// @Param message body createAPIKeyRequest true "api key request body"
// @Success 200 {object} createdAPIKeyResponse
// @Failure 400 {object} APIStatus
// @Failure 401 {object} APIStatus
// @Failure 404 {object} APIStatus
// @Failure 500 {object} APIStatus
// @Router /developers/keys [post]
func (d *apiKeyHandler) create(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("creating api key")
	req := new(createAPIKeyRequest)
	organization := getOrganizationFromContext(r.Context())
	user := getUserFromContext(r.Context())

	if err := render.Bind(r, req); err != nil {
		return newAPIStatus(http.StatusBadRequest, "invalid request body"), StatusFailed
	}
	if err := req.Validate(); err != nil {
		return newAPIStatus(http.StatusBadRequest, err.Error()), StatusFailed
	}

	existingAPIKey, err := d.apiKeyRepo.FetchByName(ctx, req.Name, organization.ID)
	if err != nil && !errors.Is(err, logbase.ErrAPIKeyNotFound) {
		logger.Error("error checking for existing api key", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "could not verify api key name"), StatusFailed
	}

	if existingAPIKey != nil {
		return newAPIStatus(http.StatusBadRequest, "api key with the same name already exists"), StatusFailed
	}

	value := util.GenerateRandom()
	encrypted := "lg_" + logbase.HashKey(d.cfg.APIKey.HashSecret, value)

	key := &logbase.APIKey{
		OrganizationID: organization.ID,
		CreatedBy:      user.ID,
		Value:          encrypted,
		Name:           req.Name,
	}

	if err := d.apiKeyRepo.Create(ctx, key); err != nil {
		logger.Error("could not create api key", zap.Error(err))
		status := http.StatusInternalServerError
		msg := "could not create api key"

		if errors.Is(err, logbase.ErrAPIKeyMaxLimit) {
			status = http.StatusBadRequest
			msg = err.Error()
		}
		return newAPIStatus(status, msg), StatusFailed
	}

	return createdAPIKeyResponse{
		APIStatus: newAPIStatus(http.StatusOK, "api key created"),
		Value:     encrypted,
	}, StatusSuccess
}

// @Description list api keys
// @Tags developers
// @Accept  json
// @Produce  json
// @Success 200 {object} listAPIKeysResponse
// @Failure 400 {object} APIStatus
// @Failure 401 {object} APIStatus
// @Failure 404 {object} APIStatus
// @Failure 500 {object} APIStatus
// @Router /developers/keys [get]
func (d *apiKeyHandler) list(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("listing api keys")

	opts := logbase.APIKeyOptions{
		OrganizationID: getOrganizationFromContext(r.Context()).ID,
	}

	keys, err := d.apiKeyRepo.List(ctx, opts)
	if err != nil {
		logger.Error("could not list keys",
			zap.Error(err))

		return newAPIStatus(http.StatusInternalServerError, "could not list api keys"), StatusFailed
	}

	return listAPIKeysResponse{
		APIStatus: newAPIStatus(http.StatusOK, "api key fetched successfully"),
		Keys:      keys,
	}, StatusSuccess
}

type revokeAPIKeyRequest struct {
	GenericRequest

	Strategy logbase.RevocationType `json:"strategy,omitempty" validate:"required"`
}

func (c *revokeAPIKeyRequest) Validate() error {
	if !c.Strategy.IsValid() {
		return errors.New("please provide a valid revocation strategy")
	}

	return nil
}

// @Description revoke a specific api key
// @Tags developers
// @Accept  json
// @Produce  json
// @Param reference path string required "api key unique reference.. e.g api_key_"
// @Param message body revokeAPIKeyRequest true "api key request body"
// @Success 200 {object} APIStatus
// @Failure 400 {object} APIStatus
// @Failure 401 {object} APIStatus
// @Failure 404 {object} APIStatus
// @Failure 500 {object} APIStatus
// @Router /developers/keys/{reference} [delete]
func (d *apiKeyHandler) revoke(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("revoking api key")

	organization := getOrganizationFromContext(r.Context())
	user := getUserFromContext(r.Context())

	if organization.ID != user.MetaData.OrganizationID {
		return newAPIStatus(http.StatusForbidden, "unauthorized"), StatusFailed
	}

	req := new(revokeAPIKeyRequest)
	if err := render.Bind(r, req); err != nil {
		return newAPIStatus(http.StatusBadRequest, "invalid request body"), StatusFailed
	}
	if err := req.Validate(); err != nil {
		return newAPIStatus(http.StatusBadRequest, err.Error()), StatusFailed
	}

	ref := chi.URLParam(r, "reference")
	if util.IsStringEmpty(ref) {
		logger.Error("invalid api key reference", zap.String("reference", ref))
		return newAPIStatus(http.StatusBadRequest, "invalid api key reference"), StatusFailed
	}

	apiKeyID, err := uuid.Parse(ref)
	if err != nil {
		logger.Error("invalid api key reference", zap.String("reference", ref), zap.Error(err))
		return newAPIStatus(http.StatusBadRequest, "invalid api key reference"), StatusFailed
	}

	opts := logbase.APIKeyOptions{
		ID: apiKeyID,
	}

	key, err := d.apiKeyRepo.Fetch(ctx, opts)
	if err != nil {
		msg := "could not fetch api key"
		status := http.StatusInternalServerError
		logger.Error("error fetching api key", zap.Error(err))

		if errors.Is(err, logbase.ErrAPIKeyNotFound) {
			msg = err.Error()
			status = http.StatusNotFound
		}

		return newAPIStatus(status, msg), StatusFailed
	}

	if key.OrganizationID != organization.ID {
		return newAPIStatus(http.StatusForbidden, "unauthorized"), StatusFailed
	}

	if key.IsRevoked() {
		return newAPIStatus(http.StatusBadRequest, "api key already revoked"), StatusFailed
	}

	opts = logbase.APIKeyOptions{
		APIKey:         key,
		RevocationType: req.Strategy,
	}

	if err := d.apiKeyRepo.Revoke(ctx, opts); err != nil {
		logger.Error("could not revoke api key", zap.Error(err))

		status := http.StatusInternalServerError
		msg := "could not revoke api key"

		if errors.Is(err, logbase.ErrAPIKeyMaxLimit) {
			status = http.StatusBadRequest
			msg = err.Error()
		}

		return newAPIStatus(status, msg), StatusFailed
	}

	return newAPIStatus(http.StatusOK, "api key revoked"), StatusSuccess
}
