package logtrace

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ENUM(succesful,failed, expired, active)
type SessionStatus string

type Session struct {
	ID             uuid.UUID      `json:"id"                bun:"type:uuid,default:uuid_generate_v4(),pk"`
	UserID         string         `json:"user_id"           bun:"user_id"`
	UserName       string         `json:"username" bun:"username"`
	RequestDetails RequestDetails `json:"request_details" bun:"type:jsonb"`
	LoginAt        time.Time      `json:"login_at"          bun:"default:current_timestamp,notnull"`
	OrganizationID uuid.UUID      `json:"organization_id"   bun:"type:uuid,nullzero"`
	LogoutAt       time.Time      `json:"logout_at"         bun:",nullzero"`
	Status         string         `json:"status"            bun:",notnull"`
	Token          string         `json:"token"             bun:"type:text"`
	Metadata       Metadata       `json:"metadata"          bun:"type:jsonb"`
	CreatedAt      time.Time      `json:"created_at"        bun:"default:current_timestamp,notnull"`
	UpdatedAt      time.Time      `json:"updated_at"        bun:"default:current_timestamp,notnull"`
	DeletedAt      *time.Time     `json:"-"      bun:",soft_delete,nullzero"`

	bun.BaseModel `json:"-" bun:"table:sessions"`
}

type FindSessionOptions struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	OrganizationID uuid.UUID
	Token          string
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
	Logout(context.Context, *FindSessionOptions) error
	Metrics(context.Context, *FindSessionOptions) (int64, int64, error)
}
