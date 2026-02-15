package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gitlab.com/logbase/logbase"
	"gitlab.com/logbase/logbase/internal/pkg/jwttoken"
	jwttoken_mocks "gitlab.com/logbase/logbase/internal/pkg/jwttoken/mocks"
	logbase_mocks "gitlab.com/logbase/logbase/mocks"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestRequireOrganizationValidSubscription(t *testing.T) {
	t.Run("skip organization validation for account me route", func(t *testing.T) {
		rr := httptest.NewRecorder()

		req := httptest.NewRequest(http.MethodGet, "/v1/auth/account/me", nil)
		req.Header.Add("Content-Type", "application/json")

		requireOrganizationValidSubscription(getConfig())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.NoError(t, json.NewEncoder(w).Encode("{}"))
		})).ServeHTTP(rr, req)

		require.Equal(t, http.StatusOK, rr.Code)
		verifyMatch(t, rr)
	})

	t.Run("selected organization required when context is missing", func(t *testing.T) {
		rr := httptest.NewRecorder()

		req := httptest.NewRequest(http.MethodGet, "/v1/events", nil)
		req.Header.Add("Content-Type", "application/json")

		requireOrganizationValidSubscription(getConfig())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.NoError(t, json.NewEncoder(w).Encode("{}"))
		})).ServeHTTP(rr, req)

		require.Equal(t, http.StatusPreconditionRequired, rr.Code)
		verifyMatch(t, rr)
	})

	t.Run("sub not active", func(t *testing.T) {
		rr := httptest.NewRecorder()

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Add("Content-Type", "application/json")
		req = req.WithContext(writeOrganizationToCtx(req.Context(), &logbase.Organization{}))

		requireOrganizationValidSubscription(getConfig())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.NoError(t, json.NewEncoder(w).Encode("{}"))
		})).ServeHTTP(rr, req)

		require.Equal(t, http.StatusPaymentRequired, rr.Code)
		verifyMatch(t, rr)
	})

	t.Run("sub active", func(t *testing.T) {
		rr := httptest.NewRecorder()

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Add("Content-Type", "application/json")
		req = req.WithContext(writeOrganizationToCtx(req.Context(), &logbase.Organization{
			IsSubscriptionActive: true,
		}))

		requireOrganizationValidSubscription(getConfig())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.NoError(t, json.NewEncoder(w).Encode("{}"))
		})).ServeHTTP(rr, req)

		require.Equal(t, http.StatusOK, rr.Code)
		verifyMatch(t, rr)
	})
}

func TestTokenFromRequest(t *testing.T) {
	tests := []struct {
		name          string
		authHeader    string
		expectedToken string
		expectError   bool
	}{
		{
			name:          "valid bearer token",
			authHeader:    "Bearer abc123",
			expectedToken: "abc123",
			expectError:   false,
		},
		{
			name:        "missing bearer prefix",
			authHeader:  "abc123",
			expectError: true,
		},
		{
			name:        "empty auth header",
			authHeader:  "",
			expectError: true,
		},
		{
			name:        "malformed bearer token",
			authHeader:  "Bearer",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			token, err := tokenFromRequest(req)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedToken, token)
			}
		})
	}
}

func TestGetIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expectedIP string
	}{
		{
			name: "cloudflare ip",
			headers: map[string]string{
				"CF-Connecting-IP": "1.2.3.4",
			},
			expectedIP: "1.2.3.4",
		},
		{
			name: "x-forwarded-for single ip",
			headers: map[string]string{
				"X-Forwarded-For": "5.6.7.8",
			},
			expectedIP: "5.6.7.8",
		},
		{
			name: "x-forwarded-for multiple ips",
			headers: map[string]string{
				"X-Forwarded-For": "9.10.11.12, 13.14.15.16",
			},
			expectedIP: "9.10.11.12",
		},
		{
			name: "x-real-ip",
			headers: map[string]string{
				"X-Real-IP": "17.18.19.20",
			},
			expectedIP: "17.18.19.20",
		},
		{
			name:       "remote addr fallback",
			remoteAddr: "21.22.23.24",
			expectedIP: "21.22.23.24",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			if tt.remoteAddr != "" {
				req.RemoteAddr = tt.remoteAddr
			}

			ip := getIP(req)
			require.Equal(t, tt.expectedIP, ip)
		})
	}
}

func TestHTTPThrottleKeyFunc(t *testing.T) {
	t.Run("authenticated user", func(t *testing.T) {
		userID := uuid.New()
		user := &logbase.User{ID: userID}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx := writeUserToCtx(req.Context(), user)
		req = req.WithContext(ctx)

		key, err := HTTPThrottleKeyFunc(req)
		require.NoError(t, err)
		require.Equal(t, userID.String(), key)
	})

	t.Run("unauthenticated user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("CF-Connecting-IP", "1.2.3.4")

		key, err := HTTPThrottleKeyFunc(req)
		require.NoError(t, err)
		require.Equal(t, "1.2.3.4", key)
	})
}

func TestContextHelpers(t *testing.T) {
	t.Run("user context", func(t *testing.T) {
		ctx := t.Context()
		user := &logbase.User{ID: uuid.New()}

		// Test writing and reading user
		ctx = writeUserToCtx(ctx, user)
		require.True(t, doesUserExistInContext(ctx))
		require.Equal(t, user, getUserFromContext(ctx))
	})

	t.Run("organization context", func(t *testing.T) {
		ctx := t.Context()
		organization := &logbase.Organization{ID: uuid.New()}

		// Test writing and reading organization
		ctx = writeOrganizationToCtx(ctx, organization)
		require.True(t, doesOrganizationExistInContext(ctx))
		require.Equal(t, organization, getOrganizationFromContext(ctx))
	})
}

func TestJsonResponse(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler := jsonResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(`{"test": true}`))
		if err != nil {
			t.Fatal(err)
		}
	}))

	handler.ServeHTTP(rr, req)

	require.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

func TestRequireAuthentication(t *testing.T) {
	logger := zap.NewNop()
	cfg := getConfig()

	t.Run("missing token", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		jwtManager := jwttoken_mocks.NewMockJWTokenManager(ctrl)
		userRepo := logbase_mocks.NewMockUserRepository(ctrl)
		orgRepo := logbase_mocks.NewMockOrganizationRepository(ctrl)

		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		handler := requireAuthentication(
			logger,
			jwtManager,
			cfg,
			userRepo,
			orgRepo,
		)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("invalid token", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		jwtManager := jwttoken_mocks.NewMockJWTokenManager(ctrl)
		userRepo := logbase_mocks.NewMockUserRepository(ctrl)
		orgRepo := logbase_mocks.NewMockOrganizationRepository(ctrl)

		jwtManager.EXPECT().
			ParseJWToken("invalid-token").
			Return(jwttoken.JWTokenData{}, errors.New("invalid token"))

		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")

		handler := requireAuthentication(
			logger,
			jwtManager,
			cfg,
			userRepo,
			orgRepo,
		)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))

		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("expired token", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userID := uuid.New()
		jwtManager := jwttoken_mocks.NewMockJWTokenManager(ctrl)
		userRepo := logbase_mocks.NewMockUserRepository(ctrl)
		orgRepo := logbase_mocks.NewMockOrganizationRepository(ctrl)

		jwtManager.EXPECT().
			ParseJWToken("invalid-token").
			Return(jwttoken.JWTokenData{UserID: userID, ExpiresAt: time.Now().Add(-time.Hour)}, nil)

		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")

		handler := requireAuthentication(
			logger,
			jwtManager,
			cfg,
			userRepo,
			orgRepo,
		)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("user not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userID := uuid.New()
		jwtManager := jwttoken_mocks.NewMockJWTokenManager(ctrl)
		userRepo := logbase_mocks.NewMockUserRepository(ctrl)
		orgRepo := logbase_mocks.NewMockOrganizationRepository(ctrl)

		jwtManager.EXPECT().
			ParseJWToken("valid-token").
			Return(jwttoken.JWTokenData{UserID: userID, ExpiresAt: time.Now().Add(time.Hour)}, nil)

		userRepo.EXPECT().
			List(gomock.Any(), &logbase.FindUserOptions{ID: userID}).
			Return(nil, errors.New("user not found"))

		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer valid-token")

		handler := requireAuthentication(
			logger,
			jwtManager,
			cfg,
			userRepo,
			orgRepo,
		)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusInternalServerError, rr.Code)
	})

	t.Run("user without organization accessing auth/connect route", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userID := uuid.New()
		jwtManager := jwttoken_mocks.NewMockJWTokenManager(ctrl)
		userRepo := logbase_mocks.NewMockUserRepository(ctrl)
		orgRepo := logbase_mocks.NewMockOrganizationRepository(ctrl)

		jwtManager.EXPECT().
			ParseJWToken("valid-token").
			Return(jwttoken.JWTokenData{UserID: userID, ExpiresAt: time.Now().Add(time.Hour)}, nil)

		userRepo.EXPECT().
			List(gomock.Any(), &logbase.FindUserOptions{ID: userID}).
			Return(&logbase.User{
				ID: userID,
				Metadata: &logbase.UserMetadata{
					OrganizationID: []uuid.UUID{}, // No organization
				},
			}, nil)

		// For auth/connect route, organization repository should never be called
		// since we check the path before attempting to fetch organization

		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/auth/connect", nil)
		req.Header.Set("Authorization", "Bearer valid-token")

		handler := requireAuthentication(
			logger,
			jwtManager,
			cfg,
			userRepo,
			orgRepo,
		)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// For auth/connect route, we should still have user in context
			user := getUserFromContext(r.Context())
			require.Equal(t, userID, user.ID)
			require.Empty(t, user.Metadata.OrganizationID)

			// Organization should not be in context since we never fetched it
			require.False(t, doesOrganizationExistInContext(r.Context()))

			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("user with organization accessing protected route", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userID := uuid.New()
		orgID := uuid.New()
		jwtManager := jwttoken_mocks.NewMockJWTokenManager(ctrl)
		userRepo := logbase_mocks.NewMockUserRepository(ctrl)
		orgRepo := logbase_mocks.NewMockOrganizationRepository(ctrl)

		jwtManager.EXPECT().
			ParseJWToken("valid-token").
			Return(jwttoken.JWTokenData{UserID: userID, ExpiresAt: time.Now().Add(time.Hour)}, nil)

		userRepo.EXPECT().
			List(gomock.Any(), &logbase.FindUserOptions{ID: userID}).
			Return(&logbase.User{
				ID: userID,
				Metadata: &logbase.UserMetadata{
					OrganizationID: []uuid.UUID{orgID},
				},
			}, nil)

		orgRepo.EXPECT().
			List(gomock.Any(), &logbase.FindOrganizationOptions{ID: orgID}).
			Return(&logbase.Organization{ID: orgID}, nil)

		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/protected", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		req.Header.Set("X-Organization-ID", orgID.String())

		handler := requireAuthentication(
			logger,
			jwtManager,
			cfg,
			userRepo,
			orgRepo,
		)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify user and organization are in context
			user := getUserFromContext(r.Context())
			require.Equal(t, userID, user.ID)

			organization := getOrganizationFromContext(r.Context())
			require.Equal(t, orgID, organization.ID)

			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("missing selected organization header for protected route", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userID := uuid.New()
		orgID := uuid.New()
		jwtManager := jwttoken_mocks.NewMockJWTokenManager(ctrl)
		userRepo := logbase_mocks.NewMockUserRepository(ctrl)
		orgRepo := logbase_mocks.NewMockOrganizationRepository(ctrl)

		jwtManager.EXPECT().
			ParseJWToken("valid-token").
			Return(jwttoken.JWTokenData{UserID: userID, ExpiresAt: time.Now().Add(time.Hour)}, nil)

		userRepo.EXPECT().
			List(gomock.Any(), &logbase.FindUserOptions{ID: userID}).
			Return(&logbase.User{
				ID: userID,
				Metadata: &logbase.UserMetadata{
					OrganizationID: []uuid.UUID{orgID},
				},
			}, nil)

		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/protected", nil)
		req.Header.Set("Authorization", "Bearer valid-token")

		handler := requireAuthentication(
			logger,
			jwtManager,
			cfg,
			userRepo,
			orgRepo,
		)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusPreconditionRequired, rr.Code)
	})
}

func TestResolveOrganizationIDForRequest(t *testing.T) {
	t.Run("fallback to only organization when header is missing", func(t *testing.T) {
		orgID := uuid.New()
		user := &logbase.User{
			Metadata: &logbase.UserMetadata{
				OrganizationID: []uuid.UUID{orgID},
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/v1/protected", nil)

		resolvedID, err := resolveOrganizationIDForRequest(req, user)
		require.NoError(t, err)
		require.Equal(t, orgID, resolvedID)
	})

	t.Run("fallback to only organization when header is stale", func(t *testing.T) {
		orgID := uuid.New()
		user := &logbase.User{
			Metadata: &logbase.UserMetadata{
				OrganizationID: []uuid.UUID{orgID},
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/v1/protected", nil)
		req.Header.Set(organizationIDHeader, uuid.NewString())

		resolvedID, err := resolveOrganizationIDForRequest(req, user)
		require.NoError(t, err)
		require.Equal(t, orgID, resolvedID)
	})

	t.Run("reject stale header when user has multiple organizations", func(t *testing.T) {
		user := &logbase.User{
			Metadata: &logbase.UserMetadata{
				OrganizationID: []uuid.UUID{uuid.New(), uuid.New()},
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/v1/protected", nil)
		req.Header.Set(organizationIDHeader, uuid.NewString())

		resolvedID, err := resolveOrganizationIDForRequest(req, user)
		require.Error(t, err)
		require.Equal(t, uuid.Nil, resolvedID)
		require.Equal(t, "you are not a member of the selected organization", err.Error())
	})
}
