package logbase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

var ErrAuditLogNotFound = LogbaseError("audit log not found")

type (
	ActionType string
)

type Metadata struct {
	Event       string `json:"event"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

type AuditLog struct {
	ID             uuid.UUID  `json:"id"          bun:"type:uuid,default:uuid_generate_v4(),pk"`
	Action         string     `json:"action"      bun:",notnull"`
	Timestamp      time.Time  `json:"timestamp"   bun:",notnull"`
	IPAddress      string     `json:"ip_address"`
	UserID         uuid.UUID  `json:"user_id"     bun:"type:uuid"`
	Metadata       *Metadata  `json:"metadata"   bun:"type:jsonb"`
	CreatedAt      time.Time  `json:"created_at"  bun:"default:current_timestamp,notnull"`
	RequestID      string     `json:"request_id"`
	OrganizationID uuid.UUID  `json:"organization_id" bun:"type:uuid"`
	UpdatedAt      time.Time  `json:"updated_at"  bun:"default:current_timestamp,notnull"`
	DeletedAt      *time.Time `json:"-,omitempty" bun:",soft_delete,nullzero"`
	bun.BaseModel  `json:"-" bun:"table:audit_logs"`
}

type FindAuditLogOptions struct {
	OrganizationID uuid.UUID
	ID             uuid.UUID
	UserID         uuid.UUID
	Paginator      Paginator
}

type AuditLogRepository interface {
	Create(context.Context, *AuditLog) error
	ListAll(context.Context, FindAuditLogOptions) ([]*AuditLog, int64, error)
	List(context.Context, *FindAuditLogOptions) (*AuditLog, error)
	Delete(context.Context, FindAuditLogOptions) error
}
