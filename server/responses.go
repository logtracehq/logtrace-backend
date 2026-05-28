package server

import (
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/google/uuid"
	"gitlab.com/logtrace/logtrace"
)

type GenericRequest struct{}

func (g GenericRequest) Bind(_ *http.Request) error { return nil }

type pagingInfo struct {
	Total   int64 `json:"total" validate:"required"`
	PerPage int64 `json:"per_page" validate:"required"`
	Page    int64 `json:"page" validate:"required"`
}

type meta struct {
	Paging pagingInfo `json:"paging" validate:"required"`
}

type APIStatus struct {
	statusCode int
	Message    string `json:"message" validate:"required"`
}

func (a APIStatus) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, a.statusCode)
	return nil
}

type APIError struct {
	APIStatus
}

func newAPIStatus(code int, s string) APIStatus {
	return APIStatus{
		statusCode: code,
		Message:    s,
	}
}

type Event struct {
	ID                  uuid.UUID         `json:"id"`
	Type                string            `json:"type"`
	Username            string            `json:"username"`
	UserID              string            `json:"user_id"`
	HTTPMethod          string            `json:"http_method"`
	HTTPStatusCode      int               `json:"http_status_code"`
	HTTPEndpoint        string            `json:"http_endpoint"`
	HTTPRequestDuration string            `json:"http_request_duration"`
	IPAddress           string            `json:"ip_address"`
	ClientUserAgent     string            `json:"client_user_agent"`
	GeoIPLocation       string            `json:"geo_ip_location"`
	ActionName          string            `json:"action_name"`
	Metadata            logtrace.Metadata `json:"metadata"`
	CreatedAt           time.Time         `json:"created_at"`
	OperatingSystem     string            `json:"operating_system"`
}
type fetchEventResponse struct {
	Event *Event `json:"event"`
	APIStatus
}

type fetchEventsResponse struct {
	Events []*Event `json:"events"`
	Meta   meta     `json:"meta"`
	APIStatus
}

type fetchPlanResponse struct {
	Plan *logtrace.Plan `json:"plan"`
	APIStatus
}

type fetchPlansResponse struct {
	Plans []logtrace.Plan `json:"plans"`
	APIStatus
}

type fetchUserResponse struct {
	User          User                    `json:"user"`
	Organization  *logtrace.Organization  `json:"organization"`
	Organizations []logtrace.Organization `json:"organizations"`
	Token         string                  `json:"token"`
	APIStatus
}
type User struct {
	ID          string                 `json:"id"`
	Username    string                 `json:"username"`
	Email       string                 `json:"email"`
	FullName    string                 `json:"full_name"`
	CreatedAt   time.Time              `json:"created_at"`
	LastLoginAt time.Time              `json:"last_login_at"`
	Metadata    *logtrace.UserMetadata `json:"metadata"`
	RoleName    string                 `json:"role_name"`
}

type Session struct {
	ID                  uuid.UUID              `json:"id"`
	UserID              string                 `json:"user_id"`
	UserName            string                 `json:"username"`
	LoginAt             time.Time              `json:"login_at"`
	LogoutAt            time.Time              `json:"logout_at"`
	Token               string                 `json:"token"`
	Status              logtrace.SessionStatus `json:"status"`
	Metadata            logtrace.Metadata      `json:"metadata"`
	CreatedAt           time.Time              `json:"created_at"`
	OperatingSystem     string                 `json:"operating_system"`
	HTTPMethod          string                 `json:"http_method"`
	HTTPStatusCode      int                    `json:"http_status_code"`
	HTTPEndpoint        string                 `json:"http_endpoint"`
	HTTPRequestDuration string                 `json:"http_request_duration"`
	IPAddress           string                 `json:"ip_address"`
	ClientUserAgent     string                 `json:"client_user_agent"`
	GeoIPLocation       string                 `json:"geo_ip_location"`
}

type fetchSessionResponse struct {
	Session *Session `json:"session"`
	APIStatus
}

type fetchAllSessionsResponse struct {
	Sessions []*Session `json:"sessions"`
	Meta     meta       `json:"meta"`
	APIStatus
}

type createdAPIKeyResponse struct {
	APIStatus
	Value string `json:"value" validate:"required"`
}

type Key struct {
	ID         uuid.UUID  `json:"id"`
	Scope      string     `json:"scope"`
	Name       string     `json:"name"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

type listAPIKeysResponse struct {
	Keys []*Key `json:"keys" validate:"required"`
	APIStatus
}

type AuditLog struct {
	ID                  uuid.UUID         `json:"id"`
	Action              string            `json:"action"`
	Timestamp           time.Time         `json:"timestamp"`
	UserID              string            `json:"user_id"`
	Description         string            `json:"description"`
	UserName            string            `json:"username"`
	Client              string            `json:"client"`
	HTTPMethod          string            `json:"http_method"`
	HTTPStatusCode      int               `json:"http_status_code"`
	HTTPEndpoint        string            `json:"http_endpoint"`
	HTTPRequestDuration string            `json:"http_request_duration"`
	IPAddress           string            `json:"ip_address"`
	ClientUserAgent     string            `json:"client_user_agent"`
	GeoIPLocation       string            `json:"geo_ip_location"`
	Metadata            logtrace.Metadata `json:"metadata"`
	CreatedAt           time.Time         `json:"created_at"`
	OperatingSystem     string            `json:"operating_system"`
}

type listAllAuditLogs struct {
	AuditLogs []*AuditLog `json:"audit_logs"`
	Meta      meta        `json:"meta"`
	APIStatus
}

type listAuditLog struct {
	AuditLog *AuditLog `json:"audit_log"`
	APIStatus
}

type listUsersResponse struct {
	Users []*logtrace.User `json:"users"`
	Meta  meta             `json:"meta"`
	APIStatus
}

type OrganizationResponse struct {
	Organization *logtrace.Organization `json:"organization"`
	APIStatus
}

type sessionMetricsResponse struct {
	Count int64 `json:"count"`
	APIStatus
	SuspiciousCount int64 `json:"suspicious_count"`
}

type eventMetricsResponse struct {
	Count int64 `json:"count"`
	APIStatus
}

type auditLogMetrics struct {
	Count int64 `json:"count"`
	APIStatus
}

type eventTopActorMetricsResponse struct {
	Actors []*logtrace.TopActorMetrics `json:"actors"`
	APIStatus
}

type fetchInvitationsResponse struct {
	Invitations []*logtrace.Invitation `json:"invitations"`
	APIStatus
}

type UserResponse struct {
	User *logtrace.User `json:"user"`
	APIStatus
}
