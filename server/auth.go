package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/badoux/checkmail"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/theopenlane/utils/passwd"
	"gitlab.com/logbase/logbase"
	"gitlab.com/logbase/logbase/config"
	"gitlab.com/logbase/logbase/internal/pkg/googleauth"
	"gitlab.com/logbase/logbase/internal/pkg/jwttoken"
	queue "gitlab.com/logbase/logbase/internal/pkg/queues"
	"gitlab.com/logbase/logbase/internal/pkg/util"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type CookieName string

type authHandler struct {
	googleCfg         googleauth.GoogleAuthProvider
	userRepo          logbase.UserRepository
	cfg               config.Config
	orgRepo           logbase.OrganizationRepository
	tokenManager      jwttoken.JWTokenManager
	queue             queue.QueueHandler
	emailVerification logbase.EmailVerificationRepository
	passwordRepo      logbase.PasswordRepository
}

type signUpRequest struct {
	GenericRequest

	FullName        string        `json:"full_name"`
	Email           logbase.Email `json:"email"`
	Password        string        `json:"password"`
	Organization    string        `json:"organization"`
	Phone           string        `json:"phone"`
	ConfirmPassword string        `json:"confirm_password"`
}

func (sr *signUpRequest) Validate() error {
	if util.IsStringEmpty(sr.FullName) {
		return errors.New("first name is required")
	}
	if util.IsStringEmpty(sr.Email.String()) {
		return errors.New("email is required")
	}
	if util.IsStringEmpty(sr.Organization) {
		return errors.New("company is required")
	}
	if err := checkmail.ValidateFormat(sr.Email.String()); err != nil {
		return errors.New("invalid email format")
	}
	if util.IsStringEmpty(sr.Password) {
		return errors.New("password is required")
	}

	if sr.Password != sr.ConfirmPassword {
		return errors.New("passwords do not match")
	}

	if passwd.Strength(sr.Password) < passwd.Moderate {
		return errors.New("password is too weak")
	}

	hashedPassword, err := logbase.HashPassword(sr.Password)
	if err != nil {
		return errors.New("could not hash password")
	}

	sr.Password = hashedPassword

	return nil
}

type loginRequest struct {
	GenericRequest
	Email    string `json:"email"`
	Password string `json:"password"`

	Code string `json:"code"`
}

func (l *loginRequest) Validate(provider string) error {
	if provider == "email" {
		if l.Email == "" {
			return errors.New("email is required")
		}
		if l.Password == "" {
			return errors.New("password is required")
		}

		if err := checkmail.ValidateFormat(l.Email); err != nil {
			return errors.New("invalid email format")
		}

		return nil
	}

	if provider == "google" {
		if l.Code == "" {
			return errors.New("code is required")
		}
		return nil
	}

	return errors.New("unsupported provider")
}

// @Description Login handles user authentication via OAuth2 providers.
// @Tags Authenticating
// @Accept json
// @Produce json
// @Param provider path string true "OAuth2 Provider (e.g., google)"
// @Param loginRequest body loginRequest true "Login Request Body"
// @Success 200 {object} fetchUserResponse "Login successful"
// @Failure 400 {object} APIStatus "Bad request or invalid credentials"
// @Failure 500 {object} APIStatus "Internal server error"
// @Router /auth/login/{provider} [post]
func (a *authHandler) login(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	provider := chi.URLParam(r, "provider")

	logger = logger.With(zap.String("provider", provider))
	span.SetAttributes(attribute.String("auth.provider", provider))
	logger.Debug("Authenticating user")

	// Validate provider
	if provider != "google" && provider != "email" {
		return newAPIStatus(http.StatusBadRequest, "unsupported provider"), StatusFailed
	}

	req := new(loginRequest)
	if err := render.Bind(r, req); err != nil {
		return newAPIStatus(http.StatusBadRequest, "invalid request body"), StatusFailed
	}

	if err := req.Validate(provider); err != nil {
		return newAPIStatus(http.StatusBadRequest, err.Error()), StatusFailed
	}

	// Route to appropriate login handler
	switch provider {
	case "google":
		return a.loginWithGoogle(ctx, logger, req)
	case "email":
		return a.loginWithEmail(ctx, logger, req)
	default:
		return newAPIStatus(http.StatusBadRequest, "unsupported provider"), StatusFailed
	}
}

func (a *authHandler) loginWithGoogle(
	ctx context.Context,
	logger *zap.Logger,
	req *loginRequest,
) (render.Renderer, Status) {
	token, err := a.googleCfg.Validate(ctx, googleauth.ValidateOptions{
		Code: req.Code,
	})
	if err != nil {
		logger.Error("could not exchange token", zap.Error(err))
		return newAPIStatus(
			http.StatusBadRequest,
			"could not verify your sign in with Google",
		), StatusFailed
	}

	u, err := a.googleCfg.User(ctx, token)
	if err != nil {
		logger.Error("could not fetch user details from google", zap.Error(err))
		return newAPIStatus(http.StatusBadRequest, "could not fetch user details from oauth2 provider"), StatusFailed
	}

	user := &logbase.User{
		Email:    u.Email,
		FullName: u.FullName,
	}

	return a.getOrCreateUser(ctx, logger, user)
}

func (a *authHandler) loginWithEmail(ctx context.Context, logger *zap.Logger, req *loginRequest,
) (render.Renderer, Status) {
	opts := &logbase.FindUserOptions{
		Email: logbase.Email(req.Email),
	}

	user, err := a.userRepo.Get(ctx, opts)
	if err != nil {
		if errors.Is(err, logbase.ErrUserNotFound) {
			logger.Debug("user not found", zap.String("email", req.Email))
			return newAPIStatus(http.StatusUnauthorized, "invalid email or password"), StatusFailed
		}
		logger.Error("error fetching user", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "an error occurred while logging in"), StatusFailed
	}

	userPassword, err := a.passwordRepo.Get(ctx, user.ID)
	if err != nil {

		logger.Error("error fetching user password", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "an error occurred while logging in"), StatusFailed
	}

	_, err = logbase.ComparePasswordAndHash(req.Password, userPassword.UserPassword)
	if err != nil {
		logger.Debug("invalid password", zap.String("email", req.Email))
		return newAPIStatus(http.StatusUnauthorized, "invalid email or password"), StatusFailed
	}

	return a.generateUserToken(user, logger)
}

func (a *authHandler) getOrCreateUser(
	ctx context.Context,
	logger *zap.Logger,
	user *logbase.User,
) (render.Renderer, Status) {
	_, err := a.userRepo.Create(ctx, user)
	if errors.Is(err, logbase.ErrUserExists) {
		user, err := a.userRepo.Get(ctx, &logbase.FindUserOptions{
			Email: user.Email,
		})
		if err != nil {
			logger.Error("an error occurred while fetching user", zap.Error(err))
			return newAPIStatus(
				http.StatusInternalServerError,
				"an error occurred while logging user into app",
			), StatusFailed
		}
		return a.generateUserToken(user, logger)
	}

	if err != nil {
		logger.Error("an error occurred while creating user", zap.Error(err))
		return newAPIStatus(
			http.StatusInternalServerError, "an error occurred while creating user"), StatusFailed
	}

	return a.generateUserToken(user, logger)
}

// @Description FetchCurrentUser retrieves the currently authenticated user's details.
// @Tags Users
// @Accept json
// @Produce json
// @Success 200 {object} fetchUserResponse "Fetched current user"
// @Failure 500 {object} APIStatus "Internal server error"
// @Router /auth/me [get]
func (a *authHandler) fetchCurrentUser(
	ctx context.Context,
	span trace.Span,
	logger *zap.Logger,
	w http.ResponseWriter,
	r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("fetching current user")

	user := getUserFromContext(ctx)

	var org *logbase.Organization = nil
	if doesOrganizationExistInContext(ctx) {
		org = getOrganizationFromContext(ctx)
	}

	orgs, err := a.orgRepo.List(ctx, user)
	if err != nil {
		logger.Error("an error occurred while fetching user organizations", zap.Error(err))
		return newAPIStatus(
			http.StatusInternalServerError,
			"an error occurred while fetching user organizations",
		), StatusFailed
	}

	return fetchUserResponse{
		User:          *user,
		Organization:  org,
		Organizations: orgs,
		APIStatus:     newAPIStatus(http.StatusOK, "fetched current user"),
	}, StatusSuccess
}

func (a *authHandler) generateUserToken(
	user *logbase.User,
	logger *zap.Logger,
) (render.Renderer, Status) {
	token, err := a.tokenManager.GenerateJWToken(jwttoken.JWTokenData{
		UserID: user.ID,
	})
	if err != nil {
		logger.Error("an error occurred while generating jwt token", zap.Error(err))
		return newAPIStatus(
			http.StatusInternalServerError,
			"an error occurred while generating jwt token",
		), StatusFailed
	}

	resp := fetchUserResponse{
		User:      *user,
		APIStatus: newAPIStatus(http.StatusOK, "login successful"),
		Token:     token.Token,
	}
	return resp, StatusSuccess
}

func (a *authHandler) emailSignUp(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("creating user ( email + password )")

	req := new(signUpRequest)

	if err := render.Bind(r, req); err != nil {
		return newAPIStatus(http.StatusBadRequest, "invalid request body"), StatusFailed
	}

	if err := req.Validate(); err != nil {
		return newAPIStatus(http.StatusBadRequest, err.Error()), StatusFailed
	}

	org := &logbase.Organization{
		Name:                 req.Organization,
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

	user := &logbase.User{
		Email:    req.Email,
		FullName: req.FullName,
		Status:   logbase.UserStatusActive.String(),
		Roles:    []logbase.UserRole{},
		MetaData: &logbase.UserMetaData{
			OrganizationID: org.ID,
			UserRole:       "admin",
		},
	}

	user, err = a.userRepo.Create(ctx, user)
	if errors.Is(err, logbase.ErrUserExists) {
		return newAPIStatus(http.StatusConflict, "Account already exists. Please use a different email"), StatusFailed
	}

	if err != nil {
		logger.Error("an error occurred while creating user account", zap.Error(err))
		return newAPIStatus(
			http.StatusInternalServerError,
			"could not create an account at this time. an error occurred",
		), StatusFailed
	}

	if user == nil {
		logger.Error("user repository returned nil user without error")
		return newAPIStatus(
			http.StatusInternalServerError,
			"could not create an account at this time. an error occurred",
		), StatusFailed
	}

	hashPassword, err := logbase.HashPassword(req.Password)
	if err != nil {
		logger.Error("failed to save passsword", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to save password"), StatusFailed
	}

	userPassword := &logbase.Password{
		UserPassword: hashPassword,
		UserID:       user.ID,
	}

	err = a.passwordRepo.Create(ctx, userPassword)
	if err != nil {
		logger.Error("failed to save password", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to save password"), StatusFailed
	}

	_ = a.sendVerificationEmail(user, logger)

	return a.generateUserToken(user, logger)
}

func (a *authHandler) sendVerificationEmail(user *logbase.User, logger *zap.Logger) error {
	if user.EmailVerifiedAt != nil {
		return nil
	}

	token, err := logbase.NewEmailVerification(user)
	if err != nil {
		logger.Error("could not generate email verification token", zap.Error(err))
		return nil
	}

	if err := a.emailVerification.Create(context.Background(), token); err != nil {
		logger.Error("could not store email verification token", zap.Error(err))
		return errors.New("could not store verification token")
	}

	return a.queue.Add(context.Background(), queue.QueueTopicVerifyEmail, queue.EmailVerificationOptions{
		UserID: user.ID,
		Token:  token.Token,
	})
}

type inviteUserRequest struct {
	GenericRequest

	Email    logbase.Email `json:"email"`
	FullName string        `json:"full_name"`
	Role     string        `json:"role"`
}

func (ir *inviteUserRequest) Validate() error {
	if util.IsStringEmpty(ir.Email.String()) {
		return errors.New("email is required")
	}
	if err := checkmail.ValidateFormat(ir.Email.String()); err != nil {
		return errors.New("invalid email format")
	}
	if util.IsStringEmpty(ir.Role) {
		return errors.New("role is required")
	}
	if util.IsStringEmpty(ir.FullName) {
		return errors.New("full name is required")
	}
	return nil
}

// @Description Invite a user by email.
// @Tags Users
// @Accept json
// @Produce json
// @Param inviteUserRequest body inviteUserRequest true "Invite User Request Body"
// @Success 200 {object} APIStatus "Invitation sent successfully"
// @Failure 400 {object} APIStatus "invalid request body"
// @Failure 500 {object} APIStatus "internal server error"
// @Router /auth/invite [post]
func (a *authHandler) inviteUserByEmail(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("inviting user by email")

	req := new(inviteUserRequest)
	if err := render.Bind(r, req); err != nil {
		return newAPIStatus(http.StatusBadRequest, "invalid request body"), StatusFailed
	}
	if err := req.Validate(); err != nil {
		return newAPIStatus(http.StatusBadRequest, err.Error()), StatusFailed
	}

	findOpts := &logbase.FindUserOptions{Email: req.Email}
	_, err := a.userRepo.Get(ctx, findOpts)
	if err == nil {
		return newAPIStatus(http.StatusConflict, "User already exists"), StatusFailed
	}
	if !errors.Is(err, logbase.ErrUserNotFound) {
		logger.Error("error checking user existence", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "could not check user existence"), StatusFailed
	}
	user := &logbase.User{
		Email:    req.Email,
		FullName: req.FullName,
		Status:   logbase.UserStatusPending.String(),
		MetaData: &logbase.UserMetaData{
			OrganizationID: getOrganizationFromContext(ctx).ID,
			UserRole:       logbase.RoleName(req.Role),
		},
	}

	_, err = a.userRepo.Create(ctx, user)
	if errors.Is(err, logbase.ErrUserExists) {
		return newAPIStatus(http.StatusConflict, "User already exists"), StatusFailed
	}

	token, err := logbase.NewEmailVerification(user)
	if err != nil {
		logger.Error("could not generate invite token", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "could not generate invite token"), StatusFailed
	}
	err = a.queue.Add(ctx, queue.QueueTopicInviteTeamMember, queue.InviteUserOptions{
		Email:        req.Email,
		Organization: getOrganizationFromContext(ctx).ID,
		Token:        token.Token,
	})
	if err != nil {
		logger.Error("failed to queue invite email", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to send invitation email"), StatusFailed
	}

	return newAPIStatus(http.StatusOK, "Invitation sent successfully"), StatusSuccess
}

// @Description List all users in the organization.
// @Tags Users
// @Accept json
// @Produce json
// @Success 200 {object} listUsersResponse "List of organization users"
// @Failure 500 {object} APIStatus "Internal server error"
// @Router /auth/me/users [get]
func (a *authHandler) listOrganizationUsers(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("listing organization users")

	org := getOrganizationFromContext(ctx)

	opts := &logbase.FindUserOptions{
		OrganizationID: org.ID,
	}

	users, totalCount, err := a.userRepo.List(ctx, opts)
	if err != nil {
		logger.Error("error listing organization users", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "could not list organization users"), StatusFailed
	}

	return listUsersResponse{
		Users: users,
		Meta: meta{
			Paging: pagingInfo{
				Total:   totalCount,
				PerPage: opts.Paginator.PerPage,
				Page:    opts.Paginator.Page,
			},
		}, APIStatus: newAPIStatus(http.StatusOK, "listed organization users"),
	}, StatusSuccess
}

// @Description Revoke a user's admin role in the organization.
// @Tags Users
// @Accept json
// @Produce json
// @Param reference path string true "User Reference (UUID)"
// @Success 200 {object} APIStatus "User role revoked successfully"
// @Failure 400 {object} APIStatus "Invalid user reference"
// @Failure 403 {object} APIStatus "User does not belong to your organization"
// @Failure 404 {object} APIStatus "User not found"
// @Failure 500 {object} APIStatus "Could not update user role"
// @Router /auth/account/{reference} [patch]
func (a *authHandler) revokeUserRole(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("revoking user role")

	ref := chi.URLParam(r, "reference")
	userID, err := uuid.Parse(ref)
	if err != nil {
		return newAPIStatus(http.StatusBadRequest, "invalid user reference"), StatusFailed
	}

	user, err := a.userRepo.Get(ctx, &logbase.FindUserOptions{
		ID: userID,
	})
	if err != nil {
		if errors.Is(err, logbase.ErrUserNotFound) {
			return newAPIStatus(http.StatusNotFound, "user not found"), StatusFailed
		}
		logger.Error("error fetching user", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "could not fetch user"), StatusFailed
	}

	org := getOrganizationFromContext(ctx)
	if user.MetaData.OrganizationID != org.ID {
		return newAPIStatus(http.StatusForbidden, "user does not belong to your organization"), StatusFailed
	}

	user.MetaData.UserRole = "member"
	err = a.userRepo.Update(ctx, user)
	if err != nil {
		logger.Error("error updating user role", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "could not update user role"), StatusFailed
	}

	return newAPIStatus(http.StatusOK, "user role revoked successfully"), StatusSuccess
}
