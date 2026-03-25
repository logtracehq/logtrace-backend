package logtrace

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ENUM(succesful,failed)
type SessionStatus string

type Session struct {
	ID             uuid.UUID  `json:"id"                bun:"type:uuid,default:uuid_generate_v4(),pk"`
	UserID         string     `json:"user_id"           bun:"user_id"`
	UserName       string     `json:"username" bun:"username"`
	LoginAt        time.Time  `json:"login_at"          bun:"default:current_timestamp,notnull"`
	OrganizationID uuid.UUID  `json:"organization_id"   bun:"type:uuid,nullzero"`
	LogoutAt       time.Time  `json:"logout_at"         bun:",nullzero"`
	DeviceInfo     string     `json:"device_info"       bun:"type:text"`
	IPAddress      string     `json:"ip_address"        bun:"type:text"`
	Location       string     `json:"location"          bun:"type:text"`
	Status         string     `json:"status"            bun:",notnull"`
	CreatedAt      time.Time  `json:"created_at"        bun:"default:current_timestamp,notnull"`
	UpdatedAt      time.Time  `json:"updated_at"        bun:"default:current_timestamp,notnull"`
	DeletedAt      *time.Time `json:"-,omitempty"      bun:",soft_delete,nullzero"`

	bun.BaseModel `json:"-" bun:"table:sessions"`
}

type FindSessionOptions struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	OrganizationID uuid.UUID
}

type ListSessionsOptions struct {
	Paginator      Paginator
	OrganizationID uuid.UUID `json:"organization_id"`
	Status         string    `json:"status"`
	StartDate      string    `json:"start_date"`
	EndDate        string    `json:"end_date"`
	Search         string    `json:"search"`
}

type SessionRepository interface {
	Create(context.Context, *Session) error
	List(context.Context, *FindSessionOptions) (*Session, error)
	ListAll(context.Context, *ListSessionsOptions) ([]*Session, int64, error)
	Metrics(context.Context, *FindSessionOptions) (int64, int64, error)
}
