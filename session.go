package logbase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Session struct {
	ID             uuid.UUID  `json:"id"                bun:"type:uuid,default:uuid_generate_v4(),pk"`
	UserID         uuid.UUID  `json:"user_id"           bun:"type:uuid,notnull"`
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
	Paginator Paginator
	Status    string
	StartDate string
	EndDate   string
}

type SessionRepository interface {
	Create(context.Context, *Session) error
	List(context.Context, *FindSessionOptions) (*Session, error)
	ListAll(context.Context, *ListSessionsOptions) ([]*Session, int64, error)
}
