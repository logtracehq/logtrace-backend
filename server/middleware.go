package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"gitlab.com/logbase/logbase"
	"gitlab.com/logbase/logbase/config"
	"gitlab.com/logbase/logbase/internal/pkg/jwttoken"
	"gitlab.com/logbase/logbase/internal/pkg/util"
	"go.uber.org/zap"
)

const (
	organizationCtx = "organization"
	userCtx         = "user"
	resourceCtx     = "resource"
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

// Key to use when setting the request ID.
type ctxKeyRequestID int

// RequestIDKey is the key that holds the unique request ID in a request context.
const RequestIDKey ctxKeyRequestID = 0

// HTTPThrottleKeyFunc throttles unauthenticated users by their IP.
// It goes through cloudflare, X-Forwarded-For and X-Real-IP to determine the
// correct IP
//
// For authenticated requests, it throttles individually instead of IP wild
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

func getResourceFromContext(ctx context.Context) *logbase.Resource {
	return ctx.Value(resourceCtx).(*logbase.Resource)
}

func getOrganizationFromContext(ctx context.Context) *logbase.Organization {
	return ctx.Value(organizationCtx).(*logbase.Organization)
}

func doesOrganizationExistInContext(ctx context.Context) bool {
	_, ok := ctx.Value(organizationCtx).(*logbase.Organization)
	return ok
}

func doesUserExistInContext(ctx context.Context) bool {
	_, ok := ctx.Value(userCtx).(*logbase.User)
	return ok
}

func writeUserToCtx(ctx context.Context, user *logbase.User) context.Context {
	return context.WithValue(ctx, userCtx, user)
}

func getUserFromContext(ctx context.Context) *logbase.User {
	return ctx.Value(userCtx).(*logbase.User)
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

// GetReqID returns a request ID from the given context if one is present.
// Returns the empty string if a request ID cannot be found.
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

func writeOrganizationToCtx(ctx context.Context, org *logbase.Organization) context.Context {
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
				r.URL.Path == "/v1/user" {
				next.ServeHTTP(w, r)
				return
			}

			org := getOrganizationFromContext(ctx)
			if !org.IsSubscriptionActive {
				_ = render.Render(w, r, newAPIStatus(http.StatusPaymentRequired,
					"Organization is not active. You need an active subscription"))
				return
			}

			resource := getResourceFromContext(ctx)
			if resource.ID == uuid.Nil {
				_ = render.Render(w, r, newAPIStatus(http.StatusBadRequest,
					"Resource not found in request context"))
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
	userRepo logbase.UserRepository,
	orgRepo logbase.OrganizationRepository,
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

			user, err := userRepo.Get(ctx, &logbase.FindUserOptions{
				ID: data.UserID,
			})
			if err != nil {
				logger.Error("could not fetch user from database", zap.Error(err))
				_ = render.Render(w, r, newAPIStatus(http.StatusInternalServerError, "an error occurred while checking user"))
				return
			}

			r = r.WithContext(writeUserToCtx(ctx, user))

			// For auth/connect path, we don't need to check organization
			if strings.HasPrefix(r.URL.Path, "/v1/auth/connect") ||
				(r.URL.Path == "/v1/organizations" && r.Method == http.MethodPost) {
				// r.URL.Path == "/v1/user" {
				next.ServeHTTP(w, r)
				return
			}

			if user.MetaData.OrganizationID == uuid.Nil {
				_ = render.Render(w, r, newAPIStatus(http.StatusPreconditionRequired, "you must be a member of a organization"))
				return
			}

			org, err := orgRepo.Get(ctx, logbase.FindOrganizationOptions{
				ID: user.MetaData.OrganizationID,
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
