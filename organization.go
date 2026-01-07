package logbase

import (
	"context"
	"time"

	"github.com/google/uuid"
)

var OrganizationNotFound = LogbaseError("Organization not found")

type Organization struct {
	ID                   uuid.UUID  `json:"id"          bun:"type:uuid,default:uuid_generate_v4(),pk"`
	Name                 string     `json:"name"        bun:"name,unique,notnull"`
	IsActive             bool       `json:"is_active"   bun:"is_active,default:true,notnull"`
	PlanName             string     `json:"plan_name"   bun:"plan_name,notnull"`
	CreatedAt            time.Time  `json:"created_at"  bun:"created_at,default:current_timestamp,notnull"`
	IsSubscriptionActive bool       `json:"is_subscription_active" bun:"default:true,notnull"`
	UpdatedAt            time.Time  `json:"updated_at"  bun:"updated_at,default:current_timestamp,notnull"`
	DeletedAt            *time.Time `json:"-,omitempty" bun:",soft_delete,nullzero"`
}

type ListOrganizationOptions struct {
	Paginator Paginator `json:"paginator"`
}

type FindOrganizationOptions struct {
	ID   uuid.UUID `json:"id,omitempty"`
	Name string    `json:"name,omitempty"`
}

type OrganizationRepository interface {
	Create(context.Context, *Organization) (*Organization, error)
	Get(context.Context, *FindOrganizationOptions) (*Organization, error)
	Update(context.Context, uuid.UUID) (*Organization, error)
	Delete(context.Context, *FindOrganizationOptions) error
	List(context.Context, *User) ([]Organization, error)
}
