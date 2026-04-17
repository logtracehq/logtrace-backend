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

type Event struct {
	ID              uuid.UUID  `json:"id"          bun:"type:uuid,default:uuid_generate_v4(),pk"`
	Type            string     `json:"type"        bun:",notnull"`
	Username        string     `json:"username"`
	UserID          string     `json:"user_id"`
	HTTPMethod      string     `json:"http_method"`
	HTTPStatus      string     `json:"http_status"`
	HTTPEndpoint    string     `json:"http_endpoint"`
	ClientIP        string     `json:"client_ip"`
	OrganizationID  uuid.UUID  `json:"organization_id" bun:"type:uuid,notnull"`
	ClientUserAgent string     `json:"client_user_agent"`
	GeoIPLocation   string     `json:"geo_ip_location"`
	ActionName      string     `json:"action_name"`
	Metadata        Metadata   `json:"metadata"    bun:"type:jsonb"`
	CreatedAt       time.Time  `json:"created_at"  bun:"default:current_timestamp,notnull"`
	UpdatedAt       time.Time  `json:"updated_at"  bun:"default:current_timestamp,notnull"`
	DeletedAt       *time.Time `json:"-,omitempty" bun:",soft_delete,nullzero"`
	bun.BaseModel   `json:"-" bun:"table:events"`
}

type ListEventOptions struct {
	ID             uuid.UUID `json:"id"`
	Action         string    `json:"action"`
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
