package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
	"gitlab.com/logtrace/logtrace"
	"gitlab.com/logtrace/logtrace/config"
	"gitlab.com/logtrace/logtrace/internal/pkg/googleauth"
	googleauth_mocks "gitlab.com/logtrace/logtrace/internal/pkg/googleauth/mocks"
	"gitlab.com/logtrace/logtrace/internal/pkg/jwttoken"
	jwttoken_mocks "gitlab.com/logtrace/logtrace/internal/pkg/jwttoken/mocks"
	logtrace_mocks "gitlab.com/logtrace/logtrace/mocks"
	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2"
)

func verifyMatch(t *testing.T, v interface{}) {
	g := goldie.New(t, goldie.WithFixtureDir("./testdata"))

	b := new(bytes.Buffer)

	if d, ok := v.(*httptest.ResponseRecorder); ok {
		_, err := io.Copy(b, d.Body)
		require.NoError(t, err)
	} else {
		err := json.NewEncoder(b).Encode(v)
		require.NoError(t, err)
	}

	g.Assert(t, t.Name(), b.Bytes())
}

func getConfig() config.Config {
	return config.Config{
		DBHost:     "localhost",
		DBPort:     "5432",
		DBUser:     "user",
		DBPassword: "password",
		DBName:     "logtrace",
		TZ:         "UTC",
		DBSSLMode:  "disable",
		Port:       "8000",
		LogLevel:   "debug",
		Logging: struct {
			Mode config.LogMode `yaml:"mode" mapstructure:"mode" json:"mode"`
		}{
			Mode: config.LogModeDev,
		},
		GoogleAuth: config.GoogleAuth{
			Code: "test-code",
		},

		HTTP: config.HTTP{
			Port: 8000,
			RateLimit: struct {
				Type              string        `yaml:"type" mapstructure:"type"`
				IsEnabled         bool          `yaml:"is_enabled" mapstructure:"is_enabled"`
				RequestsPerMinute int           `yaml:"requests_per_minute" mapstructure:"requests_per_minute"`
				BurstInterval     time.Duration `yaml:"burst_interval" mapstructure:"burst_interval"`
			}{
				Type:              "token_bucket",
				IsEnabled:         false,
				RequestsPerMinute: 60,
				BurstInterval:     time.Second,
			},
			Swagger: struct {
				Port int `yaml:"port" mapstructure:"port"`
			}{
				Port: 8001,
			},
			Metrics: struct {
				Enabled  bool   `yaml:"enabled" mapstructure:"enabled"`
				Username string `yaml:"username" mapstructure:"username"`
				Password string `yaml:"password" mapstructure:"password"`
			}{
				Enabled:  false,
				Username: "",
				Password: "",
			},
		},

		Otel: config.Otel{
			Endpoint: "/endpoint",
			UseTLS:   true,
			Headers:  "Header",
		},

		Auth: config.Auth{
			Google: struct {
				ClientID     string   `yaml:"client_id" mapstructure:"client_id"`
				ClientSecret string   `yaml:"client_secret" mapstructure:"client_secret"`
				RedirectURL  string   `yaml:"redirect_url" mapstructure:"redirect_url"`
				Scopes       []string `yaml:"scopes" mapstructure:"scopes"`
			}{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				RedirectURL:  "https://example.com/oauth/google",
				Scopes:       []string{"email", "profile"},
			},
			JWT: config.JWT{
				Key:      "a907e75f80910f5dc5b8c677de1de611ffa80be9d7d9f9dd614c8c7846db1062",
				Audience: "logtrace",
			},
		},

		Database: config.Database{
			Postgres: struct {
				DSN          string        `yaml:"dsn" mapstructure:"dsn"`
				LogQueries   bool          `yaml:"log_queries" mapstructure:"log_queries"`
				QueryTimeout time.Duration `yaml:"query_timeout" mapstructure:"query_timeout"`
			}{
				DSN:          "postgres://user:pass@localhost:5432/db?sslmode=disable",
				LogQueries:   true,
				QueryTimeout: 3 * time.Second,
			},
			Redis: struct {
				DSN string `yaml:"dsn" mapstructure:"dsn"`
			}{
				DSN: "redis://localhost:6379",
			},
		},

		Frontend: config.Frontend{
			AppURL: "https://app.example.com",
		},

		Email: config.Email{
			Provider:   "ses",
			Sender:     logtrace.Email("test@example.com"),
			SenderName: "Test Sender",
			SES: struct {
				Region    string `yaml:"region" mapstructure:"region"`
				AccessKey string `yaml:"access_key" mapstructure:"access_key"`
				SecretKey string `yaml:"secret_key" mapstructure:"secret_key"`
			}{
				Region:    "us-east-1",
				AccessKey: "test-access-key",
				SecretKey: "test-secret-key",
			},
		},

		APIKey: config.APIKey{
			HashSecret: "1234597u8tysdhfjhfk",
		},

		Billing: config.Billing{
			TrialDays: 14,
		},
	}
}

func getFetchCurrentUserData() []struct {
	name               string
	mockFn             func(orgRepo *logtrace_mocks.MockOrganizationRepository)
	expectedStatusCode int
	addOrganization    bool
} {
	return []struct {
		name               string
		mockFn             func(orgRepo *logtrace_mocks.MockOrganizationRepository)
		expectedStatusCode int
		addOrganization    bool
	}{
		{
			name: "could not list organizations",
			mockFn: func(orgRepo *logtrace_mocks.MockOrganizationRepository) {
				orgRepo.EXPECT().ListAll(gomock.Any(), gomock.Any()).
					Times(1).
					Return(nil, int64(0), errors.New("could not list organizations"))
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name: "listed organizations",
			mockFn: func(orgRepo *logtrace_mocks.MockOrganizationRepository) {
				orgRepo.EXPECT().ListAll(gomock.Any(), gomock.Any()).
					Times(1).
					Return([]logtrace.Organization{}, int64(0), nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "listed organizations with current organization",
			mockFn: func(orgRepo *logtrace_mocks.MockOrganizationRepository) {
				orgRepo.EXPECT().ListAll(gomock.Any(), gomock.Any()).
					Times(1).
					Return([]logtrace.Organization{}, int64(0), nil)
			},
			expectedStatusCode: http.StatusOK,
		},
	}
}

func TestSignUpRequest_Validate_DoesNotMutatePassword(t *testing.T) {
	req := signUpRequest{
		FullName:        "Test User",
		Email:           logtrace.Email("test@example.com"),
		Password:        "StrongPassword123!",
		ConfirmPassword: "StrongPassword123!",
		Organization:    "Test Company",
	}

	require.NoError(t, req.Validate())
	require.Equal(t, "StrongPassword123!", req.Password)
}

func TestUpdateUserRequest_Validate_CurrentPasswordRequiredForPasswordChange(t *testing.T) {
	req := updateUserRequest{
		FullName:        "Updated User",
		Password:        "StrongPassword123!",
		CurrentPassword: "",
	}

	require.EqualError(t, req.Validate(), "current password is required to change password")
}

func TestAuthHandler_EmailSignup(t *testing.T) {
	for _, v := range generateEmailSignupTestTable() {
		t.Run(v.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			userRepo := logtrace_mocks.NewMockUserRepository(controller)
			tokenManager := jwttoken_mocks.NewMockJWTokenManager(controller)
			emailVerification := logtrace_mocks.NewMockEmailVerificationRepository(controller)
			queueMock := logtrace_mocks.NewMockQueueHandler(controller)
			orgRepo := logtrace_mocks.NewMockOrganizationRepository(controller)
			passwordRepo := logtrace_mocks.NewMockPasswordRepository(controller)

			v.mockFn(userRepo, tokenManager, emailVerification, queueMock, orgRepo, passwordRepo)

			a := &authHandler{
				cfg:               getConfig(),
				userRepo:          userRepo,
				tokenManager:      tokenManager,
				emailVerification: emailVerification,
				queue:             queueMock,
				orgRepo:           orgRepo,
				passwordRepo:      passwordRepo,
			}

			b := bytes.NewBuffer(nil)

			require.NoError(t, json.NewEncoder(b).Encode(&v.req))

			rr := httptest.NewRecorder()

			req := httptest.NewRequest(http.MethodPost, "/", b)
			ctx := chi.NewRouteContext()
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))
			req.Header.Add("Content-Type", "application/json")

			WrapLogtraceHTTPHandler(getLogger(t), a.emailSignUp, getConfig(), "Auth.emailSignup").
				ServeHTTP(rr, req)

			require.Equal(t, v.expectedStatusCode, rr.Code)
			verifyMatch(t, rr)
		})
	}
}

func TestAuthHandler_FetchCurrentUser(t *testing.T) {
	for _, v := range getFetchCurrentUserData() {
		t.Run(v.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			orgRepo := logtrace_mocks.NewMockOrganizationRepository(controller)

			v.mockFn(orgRepo)

			a := &authHandler{
				cfg:     getConfig(),
				orgRepo: orgRepo,
			}

			b := bytes.NewBuffer(nil)

			rr := httptest.NewRecorder()

			req := httptest.NewRequest(http.MethodPost, "/", b)
			ctx := chi.NewRouteContext()
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))
			req.Header.Add("Content-Type", "application/json")

			req = req.WithContext(writeUserToCtx(req.Context(), &logtrace.User{}))

			if v.addOrganization {
				req = req.WithContext(writeOrganizationToCtx(req.Context(), &logtrace.Organization{}))
			}

			WrapLogtraceHTTPHandler(getLogger(t), a.fetchCurrentUser, getConfig(), "Auth.fetchCurrentUser").
				ServeHTTP(rr, req)

			require.Equal(t, v.expectedStatusCode, rr.Code)
			verifyMatch(t, rr)
		})
	}
}

func generateEmailSignupTestTable() []struct {
	name               string
	mockFn             func(userRepo *logtrace_mocks.MockUserRepository, tokenManager *jwttoken_mocks.MockJWTokenManager, emailVerification *logtrace_mocks.MockEmailVerificationRepository, queueMock *logtrace_mocks.MockQueueHandler, orgRepo *logtrace_mocks.MockOrganizationRepository, passwordRepo *logtrace_mocks.MockPasswordRepository)
	expectedStatusCode int
	req                signUpRequest
} {
	userID := uuid.MustParse("37f41afb-afff-45cc-bcc0-71249d95df90")

	return []struct {
		name               string
		mockFn             func(userRepo *logtrace_mocks.MockUserRepository, tokenManager *jwttoken_mocks.MockJWTokenManager, emailVerification *logtrace_mocks.MockEmailVerificationRepository, queueMock *logtrace_mocks.MockQueueHandler, orgRepo *logtrace_mocks.MockOrganizationRepository, passwordRepo *logtrace_mocks.MockPasswordRepository)
		expectedStatusCode int
		req                signUpRequest
	}{
		{
			name: "empty full name",
			mockFn: func(userRepo *logtrace_mocks.MockUserRepository, tokenManager *jwttoken_mocks.MockJWTokenManager, emailVerification *logtrace_mocks.MockEmailVerificationRepository, queueMock *logtrace_mocks.MockQueueHandler, orgRepo *logtrace_mocks.MockOrganizationRepository, passwordRepo *logtrace_mocks.MockPasswordRepository) {
				userRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatusCode: http.StatusBadRequest,
			req: signUpRequest{
				FullName:        "",
				Email:           logtrace.Email("test@example.com"),
				Password:        "StrongPassword123!",
				ConfirmPassword: "StrongPassword123!",
				Organization:    "Test Company",
			},
		},
		{
			name: "empty email",
			mockFn: func(userRepo *logtrace_mocks.MockUserRepository, tokenManager *jwttoken_mocks.MockJWTokenManager, emailVerification *logtrace_mocks.MockEmailVerificationRepository, queueMock *logtrace_mocks.MockQueueHandler, orgRepo *logtrace_mocks.MockOrganizationRepository, passwordRepo *logtrace_mocks.MockPasswordRepository) {
				userRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatusCode: http.StatusBadRequest,
			req: signUpRequest{
				FullName:        "Test User",
				Email:           logtrace.Email(""),
				Password:        "StrongPassword123!",
				ConfirmPassword: "StrongPassword123!",
				Organization:    "Test Company",
			},
		},
		{
			name: "invalid email",
			mockFn: func(userRepo *logtrace_mocks.MockUserRepository, tokenManager *jwttoken_mocks.MockJWTokenManager, emailVerification *logtrace_mocks.MockEmailVerificationRepository, queueMock *logtrace_mocks.MockQueueHandler, orgRepo *logtrace_mocks.MockOrganizationRepository, passwordRepo *logtrace_mocks.MockPasswordRepository) {
				userRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatusCode: http.StatusBadRequest,
			req: signUpRequest{
				FullName:        "Test User",
				Email:           logtrace.Email("invalid-email"),
				Password:        "StrongPassword123!",
				ConfirmPassword: "StrongPassword123!",
				Organization:    "Test Company",
			},
		},
		{
			name: "empty password",
			mockFn: func(userRepo *logtrace_mocks.MockUserRepository, tokenManager *jwttoken_mocks.MockJWTokenManager, emailVerification *logtrace_mocks.MockEmailVerificationRepository, queueMock *logtrace_mocks.MockQueueHandler, orgRepo *logtrace_mocks.MockOrganizationRepository, passwordRepo *logtrace_mocks.MockPasswordRepository) {
				userRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatusCode: http.StatusBadRequest,
			req: signUpRequest{
				FullName:        "Test User",
				Email:           logtrace.Email("test@example.com"),
				Password:        "",
				ConfirmPassword: "",
				Organization:    "Test Company",
			},
		},
		{
			name: "weak password",
			mockFn: func(userRepo *logtrace_mocks.MockUserRepository, tokenManager *jwttoken_mocks.MockJWTokenManager, emailVerification *logtrace_mocks.MockEmailVerificationRepository, queueMock *logtrace_mocks.MockQueueHandler, orgRepo *logtrace_mocks.MockOrganizationRepository, passwordRepo *logtrace_mocks.MockPasswordRepository) {
				userRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatusCode: http.StatusBadRequest,
			req: signUpRequest{
				FullName:        "Test User",
				Email:           logtrace.Email("test@example.com"),
				Password:        "weak",
				ConfirmPassword: "weak",
				Organization:    "Test Company",
			},
		},
		{
			name: "user already exists",
			mockFn: func(userRepo *logtrace_mocks.MockUserRepository, tokenManager *jwttoken_mocks.MockJWTokenManager, emailVerification *logtrace_mocks.MockEmailVerificationRepository, queueMock *logtrace_mocks.MockQueueHandler, orgRepo *logtrace_mocks.MockOrganizationRepository, passwordRepo *logtrace_mocks.MockPasswordRepository) {
				orgRepo.EXPECT().List(gomock.Any(), gomock.Any()).Times(1).Return(&logtrace.Organization{}, nil)
				orgRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(1).Return(&logtrace.Organization{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111")}, nil)
				userRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(1).Return(nil, logtrace.ErrUserExists)
			},
			expectedStatusCode: http.StatusConflict,
			req: signUpRequest{
				FullName:        "Test User",
				Email:           logtrace.Email("test@example.com"),
				Password:        "StrongPassword123!",
				ConfirmPassword: "StrongPassword123!",
				Organization:    "Test Company",
			},
		},
		{
			name: "could not create user",
			mockFn: func(userRepo *logtrace_mocks.MockUserRepository, tokenManager *jwttoken_mocks.MockJWTokenManager, emailVerification *logtrace_mocks.MockEmailVerificationRepository, queueMock *logtrace_mocks.MockQueueHandler, orgRepo *logtrace_mocks.MockOrganizationRepository, passwordRepo *logtrace_mocks.MockPasswordRepository) {
				orgRepo.EXPECT().List(gomock.Any(), gomock.Any()).Times(1).Return(&logtrace.Organization{}, nil)
				orgRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(1).Return(&logtrace.Organization{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111")}, nil)
				userRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(1).Return(nil, errors.New("db error"))
			},
			expectedStatusCode: http.StatusInternalServerError,
			req: signUpRequest{
				FullName:        "Test User",
				Email:           logtrace.Email("test@example.com"),
				Password:        "StrongPassword123!",
				ConfirmPassword: "StrongPassword123!",
				Organization:    "Test Company",
			},
		},
		{
			name: "could not generate token",
			mockFn: func(userRepo *logtrace_mocks.MockUserRepository, tokenManager *jwttoken_mocks.MockJWTokenManager, emailVerification *logtrace_mocks.MockEmailVerificationRepository, queueMock *logtrace_mocks.MockQueueHandler, orgRepo *logtrace_mocks.MockOrganizationRepository, passwordRepo *logtrace_mocks.MockPasswordRepository) {
				orgRepo.EXPECT().List(gomock.Any(), gomock.Any()).Times(1).Return(&logtrace.Organization{}, nil)
				orgRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(1).Return(&logtrace.Organization{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111")}, nil)
				userRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(1).Return(&logtrace.User{ID: userID}, nil)
				passwordRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(1).Return(nil)
				emailVerification.EXPECT().Create(gomock.Any(), gomock.Any()).Times(1).Return(nil)
				queueMock.EXPECT().Add(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)
				tokenManager.EXPECT().GenerateJWToken(gomock.Any()).Times(1).Return(jwttoken.JWTokenData{}, errors.New("token error"))
			},
			expectedStatusCode: http.StatusInternalServerError,
			req: signUpRequest{
				FullName:        "Test User",
				Email:           logtrace.Email("test@example.com"),
				Password:        "StrongPassword123!",
				ConfirmPassword: "StrongPassword123!",
				Organization:    "Test Company",
			},
		},
		{
			name: "verification email creation failed",
			mockFn: func(userRepo *logtrace_mocks.MockUserRepository, tokenManager *jwttoken_mocks.MockJWTokenManager, emailVerification *logtrace_mocks.MockEmailVerificationRepository, queueMock *logtrace_mocks.MockQueueHandler, orgRepo *logtrace_mocks.MockOrganizationRepository, passwordRepo *logtrace_mocks.MockPasswordRepository) {
				orgRepo.EXPECT().List(gomock.Any(), gomock.Any()).Times(1).Return(&logtrace.Organization{}, nil)
				orgRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(1).Return(&logtrace.Organization{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111")}, nil)
				userRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(1).Return(&logtrace.User{ID: userID}, nil)
				passwordRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(1).Return(nil)
				emailVerification.EXPECT().Create(gomock.Any(), gomock.Any()).Times(1).Return(errors.New("email verification store failed"))
			},
			expectedStatusCode: http.StatusInternalServerError,
			req: signUpRequest{
				FullName:        "Test User",
				Email:           logtrace.Email("test@example.com"),
				Password:        "StrongPassword123!",
				ConfirmPassword: "StrongPassword123!",
				Organization:    "Test Company",
			},
		},
		{
			name: "user created successfully",
			mockFn: func(userRepo *logtrace_mocks.MockUserRepository, tokenManager *jwttoken_mocks.MockJWTokenManager, emailVerification *logtrace_mocks.MockEmailVerificationRepository,
				queueMock *logtrace_mocks.MockQueueHandler, orgRepo *logtrace_mocks.MockOrganizationRepository, passwordRepo *logtrace_mocks.MockPasswordRepository,
			) {
				orgRepo.EXPECT().List(gomock.Any(), gomock.Any()).Times(1).Return(&logtrace.Organization{}, nil)
				orgRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(1).Return(&logtrace.Organization{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111")}, nil)
				userRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(1).Return(&logtrace.User{ID: userID}, nil)
				passwordRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(1).Return(nil)
				emailVerification.EXPECT().Create(gomock.Any(), gomock.Any()).Times(1).Return(nil)
				queueMock.EXPECT().Add(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)
				tokenManager.EXPECT().GenerateJWToken(gomock.Any()).Times(1).Return(jwttoken.JWTokenData{
					Token:  "test-token",
					UserID: userID,
				}, nil)
			},
			expectedStatusCode: http.StatusOK,
			req: signUpRequest{
				FullName:        "Test User",
				Email:           logtrace.Email("test@example.com"),
				Password:        "StrongPassword123!",
				ConfirmPassword: "StrongPassword123!",
				Organization:    "Test Company",
			},
		},
	}
}

func TestAuthHandler_Login(t *testing.T) {
	for _, v := range generateLoginTestTable() {
		t.Run(v.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			googleCfg := googleauth_mocks.NewMockGoogleAuthProvider(controller)
			userRepo := logtrace_mocks.NewMockUserRepository(controller)

			jwtMock := jwttoken_mocks.NewMockJWTokenManager(controller)

			v.mockFn(googleCfg, userRepo)

			a := &authHandler{
				cfg:          getConfig(),
				googleCfg:    googleCfg,
				userRepo:     userRepo,
				tokenManager: jwtMock,
			}

			b := bytes.NewBuffer(nil)

			require.NoError(t, json.NewEncoder(b).Encode(&v.req))

			rr := httptest.NewRecorder()

			req := httptest.NewRequest(http.MethodPost, "/", b)
			ctx := chi.NewRouteContext()
			ctx.URLParams.Add("provider", v.provider)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))
			req.Header.Add("Content-Type", "application/json")

			if v.expectedStatusCode == http.StatusOK {
				jwtMock.EXPECT().
					GenerateJWToken(gomock.Any()).
					Times(1).
					Return(jwttoken.JWTokenData{
						Token:  "b622268d-4512-4e3c-98da-88097753d4b9",
						UserID: uuid.MustParse("7e6ad0c8-7a96-4add-a270-52615bd808e6"),
					}, nil)
			}

			WrapLogtraceHTTPHandler(getLogger(t), a.login, getConfig(), "Auth.Login").
				ServeHTTP(rr, req)

			require.Equal(t, v.expectedStatusCode, rr.Code)
			verifyMatch(t, rr)
		})
	}
}

func generateLoginTestTable() []struct {
	name               string
	mockFn             func(googleMock *googleauth_mocks.MockGoogleAuthProvider, userRepo *logtrace_mocks.MockUserRepository)
	expectedStatusCode int
	req                loginRequest
	provider           string
} {
	reusedID := uuid.MustParse("37f41afb-afff-45cc-bcc0-71249d95df90")

	return []struct {
		name               string
		mockFn             func(googleMock *googleauth_mocks.MockGoogleAuthProvider, userRepo *logtrace_mocks.MockUserRepository)
		expectedStatusCode int
		req                loginRequest
		provider           string
	}{
		{
			name: "no code to exchange provided",
			mockFn: func(googleMock *googleauth_mocks.MockGoogleAuthProvider, userRepo *logtrace_mocks.MockUserRepository) {
				googleMock.EXPECT().
					Validate(gomock.Any(), gomock.Any()).
					Times(0)

				userRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			provider:           "google",
			expectedStatusCode: http.StatusBadRequest,
			req:                loginRequest{},
		},
		{
			name: "token exchange fails",
			mockFn: func(googleMock *googleauth_mocks.MockGoogleAuthProvider, userRepo *logtrace_mocks.MockUserRepository) {
				googleMock.EXPECT().
					Validate(gomock.Any(), googleauth.ValidateOptions{
						Code: "invalid-token",
					}).
					Times(1).
					Return(nil, errors.New("could not valdate token"))

				userRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectedStatusCode: http.StatusBadRequest,
			req: loginRequest{
				Code: "invalid-token",
			},
			provider: "google",
		},
		{
			name: "could not fetch user details",
			mockFn: func(googleMock *googleauth_mocks.MockGoogleAuthProvider, userRepo *logtrace_mocks.MockUserRepository) {
				googleMock.EXPECT().
					Validate(gomock.Any(), googleauth.ValidateOptions{
						Code: "token",
					}).
					Times(1).
					Return(&oauth2.Token{
						AccessToken: "access-token",
					}, nil)

				googleMock.EXPECT().
					User(gomock.Any(), gomock.Any()).
					Times(1).
					Return(googleauth.User{}, errors.New("could not fetch user"))

				userRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectedStatusCode: http.StatusBadRequest,
			req: loginRequest{
				Code: "token",
			},
			provider: "google",
		},
		{
			name: "duplicate email. user gets logged in inside but could not fetch details from db",
			mockFn: func(googleMock *googleauth_mocks.MockGoogleAuthProvider, userRepo *logtrace_mocks.MockUserRepository) {
				googleMock.EXPECT().
					Validate(gomock.Any(), googleauth.ValidateOptions{
						Code: "token",
					}).
					Times(1).
					Return(&oauth2.Token{
						AccessToken: "access-token",
					}, nil)

				user := googleauth.User{
					Email:    "test@test.com",
					FullName: "TEST TEST",
				}

				googleMock.EXPECT().
					User(gomock.Any(), gomock.Any()).
					Times(1).
					Return(user, nil)

				userRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(1).
					Return(nil, logtrace.ErrUserExists)

				userRepo.EXPECT().List(gomock.Any(), gomock.Any()).
					Times(1).
					Return(nil, errors.New("could not fetch user"))
			},
			expectedStatusCode: http.StatusInternalServerError,
			req: loginRequest{
				Code: "token",
			},
			provider: "google",
		},
		{
			name: "duplicate email. user gets logged in",
			mockFn: func(googleMock *googleauth_mocks.MockGoogleAuthProvider, userRepo *logtrace_mocks.MockUserRepository) {
				googleMock.EXPECT().
					Validate(gomock.Any(), googleauth.ValidateOptions{
						Code: "token",
					}).
					Times(1).
					Return(&oauth2.Token{
						AccessToken: "access-token",
					}, nil)

				user := googleauth.User{
					Email:    "test@test.com",
					FullName: "TEST TEST",
				}

				googleMock.EXPECT().
					User(gomock.Any(), gomock.Any()).
					Times(1).
					Return(user, nil)

				userRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(1).
					Return(nil, logtrace.ErrUserExists)

				userRepo.EXPECT().List(gomock.Any(), gomock.Any()).
					Times(1).
					Return(&logtrace.User{
						ID: reusedID,
					}, nil)
			},
			expectedStatusCode: http.StatusOK,
			req: loginRequest{
				Code: "token",
			},
			provider: "google",
		},
		{
			name: "could not create user in datastore",
			mockFn: func(googleMock *googleauth_mocks.MockGoogleAuthProvider, userRepo *logtrace_mocks.MockUserRepository) {
				googleMock.EXPECT().
					Validate(gomock.Any(), googleauth.ValidateOptions{
						Code: "token",
					}).
					Times(1).
					Return(&oauth2.Token{
						AccessToken: "access-token",
					}, nil)

				user := googleauth.User{
					Email:    "test@test.com",
					FullName: "JOhn Doe",
				}

				googleMock.EXPECT().
					User(gomock.Any(), gomock.Any()).
					Times(1).
					Return(user, nil)

				userRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(1).
					Return(nil, errors.New("unknown error"))
			},
			expectedStatusCode: http.StatusInternalServerError,
			req: loginRequest{
				Code: "token",
			},
			provider: "google",
		},
		{
			name: "user was succesfully created",
			mockFn: func(googleMock *googleauth_mocks.MockGoogleAuthProvider, userRepo *logtrace_mocks.MockUserRepository) {
				googleMock.EXPECT().
					Validate(gomock.Any(), googleauth.ValidateOptions{
						Code: "token",
					}).
					Times(1).
					Return(&oauth2.Token{
						AccessToken: "access-token",
					}, nil)

				user := googleauth.User{
					Email:    "test@test.com",
					FullName: "TEST TEST",
				}

				googleMock.EXPECT().
					User(gomock.Any(), gomock.Any()).
					Times(1).
					Return(user, nil)

				userRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(1).
					Return(nil, nil)
			},
			expectedStatusCode: http.StatusOK,
			req: loginRequest{
				Code: "token",
			},
			provider: "google",
		},
	}
}
