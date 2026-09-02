package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/badoux/checkmail"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/theopenlane/utils/passwd"
	"gitlab.com/logtrace/logtrace"
	"gitlab.com/logtrace/logtrace/config"
	"gitlab.com/logtrace/logtrace/internal/pkg/googleauth"
	"gitlab.com/logtrace/logtrace/internal/pkg/jwttoken"
	queue "gitlab.com/logtrace/logtrace/internal/pkg/queues"
	"gitlab.com/logtrace/logtrace/internal/pkg/util"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type CookieName string

type authHandler struct {
	googleCfg         googleauth.GoogleAuthProvider
	userRepo          logtrace.UserRepository
	cfg               config.Config
	orgRepo           logtrace.OrganizationRepository
	tokenManager      jwttoken.JWTokenManager
	queue             queue.QueueHandler
	emailVerification logtrace.EmailVerificationRepository
	passwordRepo      logtrace.PasswordRepository
}

type signUpRequest struct {
	GenericRequest

	FullName        string         `json:"full_name"`
	Email           logtrace.Email `json:"email"`
	Password        string         `json:"password"`
	Organization    string         `json:"organization"`
	Phone           string         `json:"phone"`
	ConfirmPassword string         `json:"confirm_password"`
}

type updateUserRequest struct {
	GenericRequest

	FullName        string `json:"full_name"`
	Password        string `json:"password"`
	CurrentPassword string `json:"current_password"`
	Phone           string `json:"phone"`
}

func (ur *updateUserRequest) Validate() error {
	if !util.IsStringEmpty(ur.Password) && util.IsStringEmpty(ur.CurrentPassword) {
		return errors.New("current password is required to change password")
	}
	if !util.IsStringEmpty(ur.Password) && passwd.Strength(ur.Password) < passwd.Moderate {
		return errors.New("password is too weak")
	}
	return nil
}

func (sr *signUpRequest) Validate() error {
	if util.IsStringEmpty(sr.FullName) {
		return errors.New("name is required")
	}
	if len(sr.FullName) < 4 {
		return errors.New("name must be at least 4 characters long")
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

	user := &logtrace.User{
		Email:    u.Email,
		FullName: u.FullName,
	}

	return a.getOrCreateUser(ctx, logger, user)
}

func (a *authHandler) loginWithEmail(ctx context.Context, logger *zap.Logger, req *loginRequest,
) (render.Renderer, Status) {
	opts := &logtrace.FindUserOptions{
		Email: logtrace.Email(req.Email),
	}

	user, err := a.userRepo.List(ctx, opts)
	if err != nil {
		if errors.Is(err, logtrace.ErrUserNotFound) {
			logger.Debug("user not found", zap.String("email", req.Email))
			return newAPIStatus(http.StatusUnauthorized, "invalid email or password"), StatusFailed
		}
		logger.Error("error fetching user", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "an error occurred while logging in"), StatusFailed
	}

	userPassword, err := a.passwordRepo.List(ctx, user.ID)
	if err != nil {

		logger.Error("error fetching user password", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "an error occurred while logging in"), StatusFailed
	}

	_, err = logtrace.ComparePasswordAndHash(req.Password, userPassword.UserPassword)
	if err != nil {
		logger.Debug("invalid password", zap.String("email", req.Email))
		return newAPIStatus(http.StatusUnauthorized, "invalid email or password"), StatusFailed
	}

	return a.generateUserToken(user, logger)
}

func (a *authHandler) getOrCreateUser(
	ctx context.Context,
	logger *zap.Logger,
	user *logtrace.User,
) (render.Renderer, Status) {
	_, err := a.userRepo.Create(ctx, user)
	if errors.Is(err, logtrace.ErrUserExists) {
		user, err := a.userRepo.List(ctx, &logtrace.FindUserOptions{
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

	var org *logtrace.Organization = nil
	if doesOrganizationExistInContext(ctx) {
		org = getOrganizationFromContext(ctx)
	}

	opts := &logtrace.FindOrganizationOptions{
		UserID: user.ID,
	}

	orgs, _, err := a.orgRepo.ListAll(ctx, opts)
	if err != nil {
		logger.Error("an error occurred while fetching user organizations", zap.Error(err))
		return newAPIStatus(
			http.StatusInternalServerError,
			"an error occurred while fetching user organizations",
		), StatusFailed
	}

	userResponse := User{
		ID:        user.ID.String(),
		Username:  user.FullName,
		Email:     user.Email.String(),
		FullName:  user.FullName,
		Phone:     user.Phone,
		CreatedAt: user.CreatedAt,
		Metadata:  user.Metadata,
	}
	if user.Metadata != nil {
		userResponse.RoleName = string(user.Metadata.UserRole)
	}

	return fetchUserResponse{
		User:          userResponse,
		Organization:  org,
		Organizations: orgs,
		APIStatus:     newAPIStatus(http.StatusOK, "fetched current user"),
	}, StatusSuccess
}

func (a authHandler) editProfile(ctx context.Context, span trace.Span, logger *zap.Logger,
	w http.ResponseWriter, r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("updating the user profile")

	currentUser := getUserFromContext(ctx)
	req := new(updateUserRequest)

	if err := render.Bind(r, req); err != nil {
		return newAPIStatus(http.StatusBadRequest, "invalid request body"), StatusFailed
	}
	if err := req.Validate(); err != nil {
		return newAPIStatus(http.StatusBadRequest, err.Error()), StatusFailed
	}

	if req.Password != "" {
		storedPassword, err := a.passwordRepo.List(ctx, currentUser.ID)
		if err != nil {
			logger.Error("error fetching user password", zap.Error(err))
			return newAPIStatus(http.StatusInternalServerError, "failed to validate current password"), StatusFailed
		}

		passwordMatches, err := logtrace.ComparePasswordAndHash(req.CurrentPassword, storedPassword.UserPassword)
		if err != nil || !passwordMatches {
			logger.Debug("current password validation failed", zap.Error(err))
			return newAPIStatus(http.StatusUnauthorized, "current password is incorrect"), StatusFailed
		}

		hashedPassword, err := logtrace.HashPassword(req.Password)
		if err != nil {
			logger.Debug("failed to hash password", zap.Error(err))
			return newAPIStatus(http.StatusInternalServerError, "failed to hash password"), StatusFailed
		}

		password := &logtrace.Password{
			UserID:       currentUser.ID,
			UserPassword: hashedPassword,
		}

		if err = a.passwordRepo.Update(ctx, password); err != nil {
			logger.Debug("failed to update the password", zap.Error(err))
			return newAPIStatus(http.StatusInternalServerError, "failed to update user password"), StatusFailed
		}
	}

	if req.FullName == "" && req.Phone == "" {
		return newAPIStatus(http.StatusOK, "User details updated successfully"), StatusSuccess
	}

	storedUser, err := a.userRepo.List(ctx, &logtrace.FindUserOptions{ID: currentUser.ID})
	if err != nil {
		logger.Error("an error occurred while fetching the user", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "Could not update user details"), StatusFailed
	}
	if req.FullName != "" {
		storedUser.FullName = req.FullName
	}
	if req.Phone != "" {
		storedUser.Phone = req.Phone
	}

	updatedUser, err := a.userRepo.Update(ctx, storedUser)
	if err != nil {
		logger.Error("an error occurred while updating the user", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError,
			"Could not update user details"), StatusFailed
	}

	return UserResponse{
		User:      updatedUser,
		APIStatus: newAPIStatus(http.StatusOK, "User details updated successfully"),
	}, StatusSuccess
}

func (a *authHandler) generateUserToken(
	user *logtrace.User,
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

	userResponse := User{
		ID:        user.ID.String(),
		Username:  user.FullName,
		Email:     user.Email.String(),
		FullName:  user.FullName,
		CreatedAt: user.CreatedAt,
		Metadata:  user.Metadata,
		RoleName:  string(user.Metadata.UserRole),
	}

	resp := fetchUserResponse{
		User:      userResponse,
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

	hashedPassword, err := logtrace.HashPassword(req.Password)
	if err != nil {
		logger.Error("failed to hash password", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to save password"), StatusFailed
	}

	twoWeeksLater := time.Now().AddDate(0, 0, 14)
	saveOrg := &logtrace.Organization{
		Name:                  req.Organization,
		IsActive:              true,
		IsSubscriptionActive:  false,
		SubscriptionExpiresAt: &twoWeeksLater,
		PlanName:              "free",
	}

	org, err := a.orgRepo.Create(ctx, saveOrg)
	if err != nil {
		if errors.Is(err, logtrace.ErrOrganizationExists) {
			logger.Error("business name is taken")
			return newAPIStatus(http.StatusConflict, "Business name is already taken"), StatusFailed
		}
		logger.Error("an error occurred while creating organization", zap.Error(err))
		return newAPIStatus(
			http.StatusInternalServerError,
			"Could not create organization at this time. an error occurred",
		), StatusFailed
	}

	user := &logtrace.User{
		Email:    req.Email,
		FullName: req.FullName,
		Status:   logtrace.UserStatusActive,
		Phone:    req.Phone,
		Roles:    []logtrace.UserRole{},
		Metadata: &logtrace.UserMetadata{
			OrganizationID: []uuid.UUID{org.ID},
			UserRole:       "admin",
		},
	}

	user, err = a.userRepo.Create(ctx, user)
	if errors.Is(err, logtrace.ErrUserExists) {
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

	userPassword := &logtrace.Password{
		UserPassword: hashedPassword,
		UserID:       user.ID,
	}

	err = a.passwordRepo.Create(ctx, userPassword)
	if err != nil {
		logger.Error("failed to save password", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to save password"), StatusFailed
	}

	if err := a.sendVerificationEmail(user, logger); err != nil {
		logger.Error("could not send verification email", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "could not send verification email"), StatusFailed
	}

	return a.generateUserToken(user, logger)
}

func (a *authHandler) sendVerificationEmail(user *logtrace.User, logger *zap.Logger) error {
	if user.EmailVerifiedAt != nil {
		return nil
	}

	token, err := logtrace.NewEmailVerification(user)
	if err != nil {
		logger.Error("could not generate email verification token", zap.Error(err))
		return err
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

	Email    logtrace.Email `json:"email"`
	FullName string         `json:"full_name"`
	Role     string         `json:"role"`
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

	findOpts := &logtrace.FindUserOptions{Email: req.Email}
	existingUser, err := a.userRepo.List(ctx, findOpts)
	if err == nil {
		if hasOrganizationMembership(existingUser, getOrganizationFromContext(ctx).ID) {
			return newAPIStatus(http.StatusConflict, "User already exists in this organization"), StatusFailed
		}
		return newAPIStatus(http.StatusConflict, "User already exists"), StatusFailed
	}
	if !errors.Is(err, logtrace.ErrUserNotFound) {
		logger.Error("error checking user existence", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "could not check user existence"), StatusFailed
	}
	user := &logtrace.User{
		Email:    req.Email,
		FullName: req.FullName,
		Status:   logtrace.UserStatusPending,
		Metadata: &logtrace.UserMetadata{
			OrganizationID: []uuid.UUID{getOrganizationFromContext(ctx).ID},
			UserRole:       logtrace.RoleName(req.Role),
		},
	}

	_, err = a.userRepo.Create(ctx, user)
	if errors.Is(err, logtrace.ErrUserExists) {
		return newAPIStatus(http.StatusConflict, "User already exists"), StatusFailed
	}

	token, err := logtrace.NewEmailVerification(user)
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

	opts := &logtrace.FindUserOptions{
		OrganizationID: org.ID,
	}

	users, totalCount, err := a.userRepo.ListAll(ctx, opts)
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

	user, err := a.userRepo.List(ctx, &logtrace.FindUserOptions{
		ID: userID,
	})
	if err != nil {
		if errors.Is(err, logtrace.ErrUserNotFound) {
			return newAPIStatus(http.StatusNotFound, "user not found"), StatusFailed
		}
		logger.Error("error fetching user", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "could not fetch user"), StatusFailed
	}

	org := getOrganizationFromContext(ctx)
	if !hasOrganizationMembership(user, org.ID) {
		return newAPIStatus(http.StatusForbidden, "user does not belong to your organization"), StatusFailed
	}

	user.Metadata.UserRole = "member"
	_, err = a.userRepo.Update(ctx, user)
	if err != nil {
		logger.Error("error updating user role", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "could not update user role"), StatusFailed
	}

	return newAPIStatus(http.StatusOK, "user role revoked successfully"), StatusSuccess
}

type resetPasswordRequest struct {
	GenericRequest
	Token           string `json:"token"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

func (r *resetPasswordRequest) Validate() error {
	if util.IsStringEmpty(r.Token) {
		return errors.New("token is required")
	}
	if util.IsStringEmpty(r.Password) {
		return errors.New("password is required")
	}
	if r.Password != r.ConfirmPassword {
		return errors.New("passwords do not match")
	}
	if passwd.Strength(r.Password) < passwd.Moderate {
		return errors.New("password is too weak")
	}
	return nil
}

type requestPasswordResetRequest struct {
	GenericRequest
	Email string `json:"email"`
}

func (r *requestPasswordResetRequest) Validate() error {
	if util.IsStringEmpty(r.Email) {
		return errors.New("email is required")
	}
	if err := checkmail.ValidateFormat(r.Email); err != nil {
		return errors.New("invalid email format")
	}
	return nil
}

// @Description RequestPasswordReset sends a password reset email to the user.
// @Tags Auth
// @Accept json
// @Produce json
// @Param requestPasswordResetRequest body requestPasswordResetRequest true "Password Reset Request Body"
// @Success 200 {object} APIStatus "Password reset email sent"
// @Failure 400 {object} APIStatus "Bad request"
// @Failure 500 {object} APIStatus "Internal server error"
// @Router /auth/password/reset [post]
func (a *authHandler) requestPasswordReset(ctx context.Context, span trace.Span, logger *zap.Logger, w http.ResponseWriter,
	r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("requesting password reset")

	req := new(requestPasswordResetRequest)
	if err := render.Bind(r, req); err != nil {
		return newAPIStatus(http.StatusBadRequest, "invalid request body"), StatusFailed
	}
	if err := req.Validate(); err != nil {
		return newAPIStatus(http.StatusBadRequest, err.Error()), StatusFailed
	}

	user, err := a.userRepo.List(ctx, &logtrace.FindUserOptions{
		Email: logtrace.Email(req.Email),
	})
	if err != nil {
		if errors.Is(err, logtrace.ErrUserNotFound) {
			logger.Debug("password reset requested for unknown email", zap.String("email", req.Email))
			return newAPIStatus(http.StatusOK, "if an account exists, a reset email will be sent"), StatusSuccess
		}
		logger.Error("error fetching user for password reset", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "an error occurred"), StatusFailed
	}

	tokenData, err := a.tokenManager.GeneratePasswordResetToken(jwttoken.JWTokenData{
		UserID: user.ID,
	})
	if err != nil {
		logger.Error("could not generate password reset token", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "could not generate reset token"), StatusFailed
	}

	if err := a.queue.Add(ctx, queue.QueueTopicResetPassword, queue.ResetPasswordOptions{
		UserID: user.ID,
		Token:  tokenData.Token,
	}); err != nil {
		logger.Error("failed to queue password reset email", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to send reset email"), StatusFailed
	}

	return newAPIStatus(http.StatusOK, "if an account exists, a reset email has been sent"), StatusSuccess
}

// @Description ResetPassword applies a new password using a valid reset token.
// @Tags Auth
// @Accept json
// @Produce json
// @Param resetPasswordRequest body resetPasswordRequest true "Reset Password Request Body"
// @Success 200 {object} APIStatus "Password reset successfully"
// @Failure 400 {object} APIStatus "Bad request or invalid/expired token"
// @Failure 500 {object} APIStatus "Internal server error"
// @Router /auth/password/confirm [post]
func (a *authHandler) resetPassword(
	ctx context.Context,
	span trace.Span,
	logger *zap.Logger,
	w http.ResponseWriter,
	r *http.Request,
) (render.Renderer, Status) {
	logger.Debug("resetting password")

	req := new(resetPasswordRequest)
	if err := render.Bind(r, req); err != nil {
		return newAPIStatus(http.StatusBadRequest, "invalid request body"), StatusFailed
	}

	if err := req.Validate(); err != nil {
		return newAPIStatus(http.StatusBadRequest, err.Error()), StatusFailed
	}

	tokenData, err := a.tokenManager.ParsePasswordResetToken(req.Token)
	if err != nil {
		logger.Debug("invalid or expired reset token", zap.Error(err))
		return newAPIStatus(http.StatusBadRequest, "invalid or expired reset token"), StatusFailed
	}

	hashedPassword, err := logtrace.HashPassword(req.Password)
	if err != nil {
		logger.Error("failed to hash password", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "failed to process new password"), StatusFailed
	}

	if err := a.passwordRepo.Update(ctx, &logtrace.Password{
		UserID:       tokenData.UserID,
		UserPassword: hashedPassword,
	}); err != nil {
		logger.Error("failed to update password", zap.Error(err))
		return newAPIStatus(http.StatusInternalServerError, "could not update password"), StatusFailed
	}

	return newAPIStatus(http.StatusOK, "password reset successfully"), StatusSuccess
}
