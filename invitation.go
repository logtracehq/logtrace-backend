package logtrace

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ENUM(pending,accepted,declined,expired,revoked)
type InvitationStatus string

type Invitation struct {
	ID             uuid.UUID        `json:"id"          bun:"type:uuid,default:uuid_generate_v4(),pk"`
	OrganizationID uuid.UUID        `json:"organization_id" bun:"type:uuid,notnull"`
	Email          Email            `json:"email" bun:"email"`
	Fullname       string           `json:"fullname"`
	Role           RoleName         `json:"role_name" bun:"column:role,"`
	Token          string           `json:"token" bun:"type:text"`
	Status         InvitationStatus `json:"status" bun:"status,default:'pending',notnull"`
	CreatedAt      time.Time        `json:"created_at"  bun:"default:current_timestamp,notnull"`
	UpdatedAt      time.Time        `json:"updated_at"  bun:"default:current_timestamp,notnull"`
	DeletedAt      *time.Time       `json:"-" bun:",soft_delete,nullzero"`
	bun.BaseModel  `json:"-" bun:"table:invitations"`
}

type ListInvitationOptions struct {
	OrganizationID uuid.UUID `json:"organization_id"`
}

type FindInvitationOptions struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	Email          string    `json:"email"`
	ID             uuid.UUID `json:"id"`
	Token          string    `json:"token"`
}

type InvitationRepository interface {
	Create(context.Context, *Invitation) error
	List(context.Context, ListInvitationOptions) ([]*Invitation, error)
	Delete(context.Context, FindInvitationOptions) error
}
