package server

import (
	"net/http"

	"github.com/go-chi/render"
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

type fetchEventResponse struct {
	Event *logbase.Event `json:"event"`
	APIStatus
}

type fetchPlanResponse struct {
	Plan *logbase.Plan
	APIStatus
}

type fetchEventsResponse struct {
	Events []*logbase.Event `json:"events"`
	Meta   meta             `json:"meta"`
	APIStatus
}

type fetchPlansResponse struct {
	Plans []logbase.Plan
	APIStatus
}

type fetchUserResponse struct {
	User          logbase.User           `json:"user"`
	Organization  *logbase.Organization  `json:"organization"`
	Organizations []logbase.Organization `json:"organizations"`
	Token         string                 `json:"token"`
	APIStatus
}

type fetchSessionResponse struct {
	Session *logbase.Session `json:"session"`
	APIStatus
}

type fetchAllSessionsResponse struct {
	Sessions []*logbase.Session `json:"sessions"`
	Meta     meta               `json:"meta"`
	APIStatus
}

type fetchResource struct {
	Resource logbase.Resource `json:"resource"`
	APIStatus
}

type createdAPIKeyResponse struct {
	APIStatus
	Value string `json:"value" validate:"required"`
}

type listAPIKeysResponse struct {
	Keys []*logbase.APIKey `json:"keys" validate:"required"`
	APIStatus
}

type fetchAllResources struct {
	Resources []*logbase.Resource `json:"resources"`
	Meta      meta                `json:"meta"`
	APIStatus
}

type listAllAuditLogs struct {
	AuditLogs []*logbase.AuditLog `json:"audit_logs"`
	Meta      meta                `json:"meta"`
	APIStatus
}

type listAuditLog struct {
	AuditLog *logbase.AuditLog `json:"audit_log"`
	APIStatus
}
