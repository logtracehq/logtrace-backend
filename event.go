package logbase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

var (
	ErrEventNotFound  = LogbaseError("Event not found")
	ErrEventsNotFound = LogbaseError("Events not found")
)

type Event struct {
	ID              uuid.UUID  `json:"id"          bun:"type:uuid,default:uuid_generate_v4(),pk"`
	Type            string     `json:"type"        bun:",notnull"`
	UserName        string     `json:"user_name"`
	HTTPMethod      string     `json:"http_method"`
	HTTPStatus      string     `json:"http_status"`
	HTTPEndpoint    string     `json:"http_endpoint"`
	ClientIP        string     `json:"client_ip"`
	OrganizationID  uuid.UUID  `json:"organization_id" bun:"type:uuid,notnull"`
	ClientUserAgent string     `json:"client_user_agent"`
	GeoIPLocation   string     `json:"geo_ip_location"`
	ActionName      string     `json:"action_name"`
	CreatedAt       time.Time  `json:"created_at"  bun:"default:current_timestamp,notnull"`
	UpdatedAt       time.Time  `json:"updated_at"  bun:"default:current_timestamp,notnull"`
	DeletedAt       *time.Time `json:"-,omitempty" bun:",soft_delete,nullzero"`
	bun.BaseModel   `json:"-" bun:"table:events"`
}

type ListEventOptions struct {
	ID             uuid.UUID
	Action         string `json:"action"`
	OrganizationID uuid.UUID
	Paginator      Paginator
	HTTPStatus     string
	HTTPMethod     string
	StartDate      string
	EndDate        string
}

type EventRepository interface {
	Create(context.Context, *Event) error
	List(context.Context, ListEventOptions) (*Event, error)
	ListAll(context.Context, *ListEventOptions) ([]*Event, int64, error)
}
