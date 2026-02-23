package server

import (
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/google/uuid"
	"gitlab.com/logbase/logbase"
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
	Message    string `json:"message," validate:"required"`
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
	ID              uuid.UUID `json:"id"`
	Type            string    `json:"type"`
	Username        string    `json:"username"`
	UserID          string    `json:"user_id"`
	HTTPMethod      string    `json:"http_method"`
	HTTPStatus      string    `json:"http_status"`
	HTTPEndpoint    string    `json:"http_endpoint"`
	ClientIP        string    `json:"client_ip"`
	OrganizationID  uuid.UUID `json:"organization_id"`
	ClientUserAgent string    `json:"client_user_agent"`
	GeoIPLocation   string    `json:"geo_ip_location"`
	ActionName      string    `json:"action_name"`
	CreatedAt       time.Time `json:"created_at"`
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
	Plan *logbase.Plan `json:"plan"`
	APIStatus
}

type fetchPlansResponse struct {
	Plans []logbase.Plan `json:"plans"`
	APIStatus
}

type fetchUserResponse struct {
	User          logbase.User           `json:"user"`
	Organization  *logbase.Organization  `json:"organization"`
	Organizations []logbase.Organization `json:"organizations"`
	Token         string                 `json:"token"`
	APIStatus
}

type Session struct {
	ID             uuid.UUID             `json:"id"`
	UserID         string                `json:"user_id"`
	UserName       string                `json:"username"`
	LoginAt        time.Time             `json:"login_at"`
	OrganizationID uuid.UUID             `json:"organization_id"`
	LogoutAt       time.Time             `json:"logout_at"`
	DeviceInfo     string                `json:"device_info"`
	IPAddress      string                `json:"ip_address"`
	Location       string                `json:"location"`
	Status         logbase.SessionStatus `json:"status"`
	CreatedAt      time.Time             `json:"created_at"`
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
	ID             uuid.UUID  `json:"id"`
	Scope          string     `json:"scope"`
	Name           string     `json:"name"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	ExpiredAt      *time.Time `json:"expired_at,omitempty"`
}

type listAPIKeysResponse struct {
	Keys []*Key `json:"keys" validate:"required"`
	APIStatus
}

type AuditLog struct {
	ID             uuid.UUID         `json:"id"`
	Action         string            `json:"action"`
	Timestamp      time.Time         `json:"timestamp"`
	IPAddress      string            `json:"ip_address"`
	UserID         string            `json:"user_id"`
	UserName       string            `json:"username"`
	CreatedAt      time.Time         `json:"created_at"`
	RequestID      string            `json:"request_id"`
	OrganizationID uuid.UUID         `json:"organization_id"`
	Metadata       *logbase.Metadata `json:"metadata"`
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
	Users []*logbase.User `json:"users"`
	Meta  meta            `json:"meta"`
	APIStatus
}

type OrganizationResponse struct {
	Organization *logbase.Organization `json:"organization"`
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

type fetchInvitationsResponse struct {
	Invitations []*logbase.Invitation `json:"invitations"`
	APIStatus
}
