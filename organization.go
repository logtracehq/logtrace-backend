package logtrace

import (
	"context"
	"time"

	"github.com/google/uuid"
)

var (
	ErrOrganizationNotFound = LogtraceError("Organization not found")
	ErrOrganizationExists   = LogtraceError("Organization already exists")
)

type Organization struct {
	ID                   uuid.UUID  `json:"id"          bun:"type:uuid,default:uuid_generate_v4(),pk"`
	Name                 string     `json:"name"        bun:"name,unique,notnull"`
	IsActive             bool       `json:"is_active"   bun:"is_active,default:true,notnull"`
	PlanName             string     `json:"plan_name"   bun:"plan_name,notnull"`
	ImageURL             string     `json:"image_url"   bun:"image_url"`
	CreatedAt            time.Time  `json:"created_at"  bun:"created_at,default:current_timestamp,notnull"`
	IsSubscriptionActive bool       `json:"is_subscription_active" bun:"default:true,notnull"`
	UpdatedAt            time.Time  `json:"updated_at"  bun:"updated_at,default:current_timestamp,notnull"`
	DeletedAt            *time.Time `json:"-,omitempty" bun:",soft_delete,nullzero"`
}

type ListOrganizationOptions struct {
	Paginator Paginator `json:"paginator"`
}

type FindOrganizationOptions struct {
	ID        uuid.UUID `json:"id,omitempty"`
	Name      string    `json:"name,omitempty"`
	UserID    uuid.UUID `json:"user_id,omitempty"`
	Plan      string    `json:"plan,omitempty"`
	Paginator Paginator `json:"paginator"`
}

type OrganizationRepository interface {
	Create(context.Context, *Organization) (*Organization, error)
	List(context.Context, FindOrganizationOptions) (*Organization, error)
	Update(context.Context, *Organization) (*Organization, error)
	Delete(context.Context, *FindOrganizationOptions) error
	ListAll(context.Context, *FindOrganizationOptions) ([]Organization, int64, error)
}
