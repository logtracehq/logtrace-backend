package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/gorilla/csrf"
	"gitlab.com/logtrace/logtrace"
	"gitlab.com/logtrace/logtrace/config"
	"gitlab.com/logtrace/logtrace/internal/pkg/jwttoken"
	"gitlab.com/logtrace/logtrace/internal/pkg/util"
	"go.uber.org/zap"
)

const (
	organizationCtx      = "organization"
	userCtx              = "user"
	organizationIDHeader = "X-Organization-ID"
)

var RequestIDHeader = "X-Request-Id"

var (
	prefix string
	reqid  uint64
)

var (
	xForwardedFor = http.CanonicalHeaderKey("X-Forwarded-For")
	xRealIP       = http.CanonicalHeaderKey("X-Real-IP")
)

type ctxKeyRequestID int

const RequestIDKey ctxKeyRequestID = 0

func HTTPThrottleKeyFunc(r *http.Request) (string, error) {
	if doesUserExistInContext(r.Context()) {
		return getUserFromContext(r.Context()).ID.String(), nil
	}

	return getIP(r), nil
}

func tokenFromRequest(r *http.Request) (string, error) {
	ss := strings.Split(r.Header.Get("Authorization"), " ")

	if len(ss) != 2 {
		return "", errors.New("invalid bearer token structure")
	}

	return ss[1], nil
}

func getOrganizationFromContext(ctx context.Context) *logtrace.Organization {
	return ctx.Value(organizationCtx).(*logtrace.Organization)
}

func doesOrganizationExistInContext(ctx context.Context) bool {
	_, ok := ctx.Value(organizationCtx).(*logtrace.Organization)
	return ok
}

func doesUserExistInContext(ctx context.Context) bool {
	_, ok := ctx.Value(userCtx).(*logtrace.User)
	return ok
}

func writeUserToCtx(ctx context.Context, user *logtrace.User) context.Context {
	return context.WithValue(ctx, userCtx, user)
}

func getUserFromContext(ctx context.Context) *logtrace.User {
	return ctx.Value(userCtx).(*logtrace.User)
}

func userOrganizationIDs(user *logtrace.User) []uuid.UUID {
	if user == nil || user.Metadata == nil {
		return nil
	}

	return user.Metadata.OrganizationID
}

func hasOrganizationMembership(user *logtrace.User, orgID uuid.UUID) bool {
	if orgID == uuid.Nil {
		return false
	}

	return slices.Contains(userOrganizationIDs(user), orgID)
}

func selectedOrganizationIDFromRequest(r *http.Request) (uuid.UUID, error) {
	selectedOrgID := strings.TrimSpace(r.Header.Get(organizationIDHeader))
	if selectedOrgID == "" {
		return uuid.Nil, errors.New("selected organization is required")
	}

	parsedID, err := uuid.Parse(selectedOrgID)
	if err != nil {
		return uuid.Nil, errors.New("invalid selected organization id")
	}

	return parsedID, nil
}

func shouldSkipOrganizationSelection(path, method string) bool {
	return strings.HasPrefix(path, "/v1/auth/connect") ||
		(path == "/v1/organizations" && method == http.MethodPost) ||
		path == "/v1/auth/account/me"
}

func resolveOrganizationIDForRequest(r *http.Request, user *logtrace.User) (uuid.UUID, error) {
	organizationIDs := userOrganizationIDs(user)
	if len(organizationIDs) == 0 {
		return uuid.Nil, errors.New("you must be a member of a organization")
	}

	selectedOrganizationID, err := selectedOrganizationIDFromRequest(r)
	if err != nil {
		if len(organizationIDs) == 1 {
			return organizationIDs[0], nil
		}

		return uuid.Nil, err
	}

	if hasOrganizationMembership(user, selectedOrganizationID) {
		return selectedOrganizationID, nil
	}

	if len(organizationIDs) == 1 {
		return organizationIDs[0], nil
	}

	return uuid.Nil, errors.New("you are not a member of the selected organization")
}

func init() {
	hostname, err := os.Hostname()
	if hostname == "" || err != nil {
		hostname = "localhost"
	}
	var buf [12]byte
	var b64 string
	for len(b64) < 10 {
		rand.Read(buf[:])
		b64 = base64.StdEncoding.EncodeToString(buf[:])
		b64 = strings.NewReplacer("+", "", "/", "").Replace(b64)
	}

	prefix = fmt.Sprintf("%s/%s", hostname, b64[0:10])
}

func RequestID(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		requestID := r.Header.Get(RequestIDHeader)
		if requestID == "" {
			myid := atomic.AddUint64(&reqid, 1)
			requestID = fmt.Sprintf("%s-%06d", prefix, myid)
		}
		ctx = context.WithValue(ctx, RequestIDKey, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}

func getIP(r *http.Request) string {
	cloudflareIP := r.Header.Get("CF-Connecting-IP")
	if !util.IsStringEmpty(cloudflareIP) {
		return cloudflareIP
	}

	if xff := r.Header.Get(xForwardedFor); xff != "" {
		i := strings.Index(xff, ", ")

		if i == -1 {
			i = len(xff)
		}

		ip := xff[:i]
		if !util.IsStringEmpty(ip) {
			return ip
		}
	}

	xIP := r.Header.Get(xRealIP)
	if !util.IsStringEmpty(xIP) {
		return xIP
	}

	return r.RemoteAddr
}

func GetReqID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok {
		return reqID
	}
	return ""
}

// NextRequestID generates the next request ID in the sequence.
func NextRequestID() uint64 {
	return atomic.AddUint64(&reqid, 1)
}

func writeRequestIDHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-ID", retrieveRequestID(r))
		next.ServeHTTP(w, r)
	})
}

func retrieveRequestID(r *http.Request) string { return middleware.GetReqID(r.Context()) }

func writeOrganizationToCtx(ctx context.Context, org *logtrace.Organization) context.Context {
	return context.WithValue(ctx, organizationCtx, org)
}

func requireOrganizationValidSubscription(
	_ config.Config,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span, _ := getTracer(r.Context(), r, "middleware.requireOrganizationValidSubscription")
			defer span.End()

			if r.URL.Path == "/v1/organizations/billing" ||
				strings.HasPrefix(r.URL.Path, "/v1/organizations/switch/") ||
				r.URL.Path == "/v1/organizations" ||
				r.URL.Path == "/v1/organizations/preferences" ||
				r.URL.Path == "/v1/user" ||
				// r.URL.Path == "/v1/plans/admin" ||
				shouldSkipOrganizationSelection(r.URL.Path, r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			if !doesOrganizationExistInContext(ctx) {
				_ = render.Render(w, r, newAPIStatus(http.StatusPreconditionRequired,
					"Organization is required"))
				return
			}

			org := getOrganizationFromContext(ctx)
			if org == nil {
				_ = render.Render(w, r, newAPIStatus(http.StatusPreconditionRequired,
					"Organization is required"))
				return
			}

			if !org.IsSubscriptionActive {
				_ = render.Render(w, r, newAPIStatus(http.StatusPaymentRequired,
					"Organization is not active. You need an active subscription"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func jsonResponse(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func requireAuthentication(
	logger *zap.Logger,
	jwtManager jwttoken.JWTokenManager,
	_ config.Config,
	userRepo logtrace.UserRepository,
	orgRepo logtrace.OrganizationRepository,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span, rid := getTracer(r.Context(), r, "middleware.requireAuthentication")
			defer span.End()

			logger := logger.With(
				zap.String("request_id", rid),
				zap.String("path", r.URL.Path),
			)

			token, err := tokenFromRequest(r)
			if err != nil {
				_ = render.Render(w, r, newAPIStatus(http.StatusUnauthorized, "session key not exists in request"))
				return
			}

			data, err := jwtManager.ParseJWToken(token)
			if err != nil {
				logger.Error("could not parse JWT", zap.Error(err))
				_ = render.Render(w, r, newAPIStatus(http.StatusUnauthorized, "could not validate JWT token"))
				return
			}

			if data.ExpiresAt.Before(time.Now()) {
				_ = render.Render(w, r, newAPIStatus(http.StatusUnauthorized, "session is expired"))
				return
			}

			user, err := userRepo.List(ctx, &logtrace.FindUserOptions{
				ID: data.UserID,
			})
			if err != nil {
				logger.Error("could not fetch user from database", zap.Error(err))
				_ = render.Render(w, r, newAPIStatus(http.StatusInternalServerError, "an error occurred while checking user"))
				return
			}

			r = r.WithContext(writeUserToCtx(ctx, user))

			if shouldSkipOrganizationSelection(r.URL.Path, r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			selectedOrganizationID, err := resolveOrganizationIDForRequest(r, user)
			if err != nil {
				statusCode := http.StatusPreconditionRequired
				if err.Error() == "you are not a member of the selected organization" {
					statusCode = http.StatusForbidden
				}

				_ = render.Render(w, r, newAPIStatus(statusCode, err.Error()))
				return
			}

			org, err := orgRepo.List(ctx, logtrace.FindOrganizationOptions{
				ID: selectedOrganizationID,
			})
			if err != nil {
				logger.Error("could not fetch organization from database", zap.Error(err))
				_ = render.Render(w, r, newAPIStatus(http.StatusInternalServerError, "an error occurred while fetching organization from database"))
				return
			}

			r = r.WithContext(writeOrganizationToCtx(r.Context(), org))
			next.ServeHTTP(w, r)
		})
	}
}

func requireAPIKeyOnly(logger *zap.Logger, _ config.Config, apiKeyRepo logtrace.APIKeyRepository,
	orgRepo logtrace.OrganizationRepository,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span, rid := getTracer(r.Context(), r, "middleware.requireAPIKeyOnly")
			defer span.End()

			logger := logger.With(
				zap.String("request_id", rid),
				zap.String("path", r.URL.Path),
				zap.Bool("is_api", true),
			)

			token := r.Header.Get("X-API-Key")
			if util.IsStringEmpty(token) {
				var err error
				token, err = tokenFromRequest(r)
				if err != nil {
					_ = render.Render(w, r, newAPIStatus(http.StatusUnauthorized, "please provide an API key in the request header"))
					return
				}
			}

			key, err := apiKeyRepo.FetchByValue(ctx, token)
			if err != nil {
				if errors.Is(err, logtrace.ErrAPIKeyNotFound) {
					_ = render.Render(w, r, newAPIStatus(http.StatusUnauthorized, "API key not found"))
					return
				}

				logger.Error("error while fetching api key", zap.Error(err))
				_ = render.Render(w, r, newAPIStatus(http.StatusInternalServerError, "An error occurred while fetching API key"))
				return
			}

			// Update last used at timestamp
			if err := apiKeyRepo.UpdateLastUsedAt(ctx, key.ID); err != nil {
				logger.Warn("failed to update api key last used at", zap.Error(err))
			}

			organization, err := orgRepo.List(ctx, logtrace.FindOrganizationOptions{
				ID: key.OrganizationID,
			})
			if err != nil {
				logger.Error("could not fetch organization from database", zap.Error(err))
				_ = render.Render(w, r, newAPIStatus(http.StatusInternalServerError, "an error occurred while fetching organization from database"))
				return
			}

			r = r.WithContext(writeOrganizationToCtx(r.Context(), organization))

			next.ServeHTTP(w, r)
		})
	}
}

func CSRFMiddleware(authKey []byte, cfg config.Config) func(http.Handler) http.Handler {
	secure := cfg.Env == config.EnvTypeProduction

	csrfMw := csrf.Protect(
		authKey,
		csrf.Secure(secure), // true in production (HTTPS)
		csrf.HttpOnly(true),
		csrf.RequestHeader("X-CSRF-Token"),
		csrf.TrustedOrigins([]string{cfg.Frontend.AppURL}),
	)

	return func(next http.Handler) http.Handler {
		return csrfMw(next)
	}
}
