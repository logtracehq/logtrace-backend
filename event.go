package logtrace

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

var (
	ErrEventNotFound  = LogtraceError("Event not found")
	ErrEventsNotFound = LogtraceError("Events not found")
)

type RequestDetails struct {
	Timestamp       time.Time         `json:"timestamp"`
	HTTPMethod      string            `json:"http_method"`
	HTTPEndpoint    string            `json:"http_endpoint"`
	HTTPStatusCode  int               `json:"http_status_code"`
	IPAddress       string            `json:"ip_address"`
	OperatingSystem string            `json:"operating_system"`
	ClientUserAgent string            `json:"client_user_agent"`
	GeoIPLocation   string            `json:"geo_ip_location"`
	RequestHeaders  map[string]string `json:"request_headers" bun:"type:jsonb"`
	RequestDuration string            `json:"request_duration"`
	RequestID       string            `json:"request_id"`
}

type Event struct {
	ID             uuid.UUID      `json:"id"          bun:"type:uuid,default:uuid_generate_v4(),pk"`
	Type           string         `json:"type"        bun:",notnull"`
	Username       string         `json:"username"`
	UserID         string         `json:"user_id"`
	OrganizationID uuid.UUID      `json:"organization_id" bun:"type:uuid,notnull"`
	RequestDetails RequestDetails `json:"request_details" bun:"type:jsonb"`
	Name           string         `json:"name"`
	Metadata       Metadata       `json:"metadata"    bun:"type:jsonb"`
	CreatedAt      time.Time      `json:"created_at"  bun:"default:current_timestamp,notnull"`
	UpdatedAt      time.Time      `json:"updated_at"  bun:"default:current_timestamp,notnull"`
	DeletedAt      *time.Time     `json:"-" bun:",soft_delete,nullzero"`

	bun.BaseModel `json:"-" bun:"table:events"`
}

type ListEventOptions struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Paginator      Paginator `json:"paginator"`
	HTTPStatus     string    `json:"http_status"`
	HTTPMethod     string    `json:"http_method"`
	StartDate      string    `json:"start_date"`
	EndDate        string    `json:"end_date"`
	Username       string    `json:"username"`
	UserID         string    `json:"user_id"`
	Search         string    `json:"search"`
}

type TopActorMetrics struct {
	Name   string `json:"name"`
	Events int64  `json:"events"`
}

type EventRepository interface {
	Create(context.Context, *Event) error
	List(context.Context, ListEventOptions) (*Event, error)
	ListAll(context.Context, *ListEventOptions) ([]*Event, int64, error)
	Metrics(context.Context, *ListEventOptions) (int64, error)
	TopActorMetrics(context.Context, *ListEventOptions) ([]*TopActorMetrics, error)
}
