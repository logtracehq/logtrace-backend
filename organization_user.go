package logtrace

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type OrganizationUser struct {
	ID             uuid.UUID  `json:"id"          bun:"type:uuid,default:uuid_generate_v4(),pk"`
	OrganizationID uuid.UUID  `json:"organization_id" bun:"type:uuid,notnull"`
	UserID         string     `json:"user_id"         bun:"user_id,notnull"`
	Username       string     `json:"username"            bun:"username, type:varchar(100),notnull"`
	Metadata       Metadata   `json:"metadata"        bun:"metadata"`
	CreatedAt      time.Time  `json:"created_at"  bun:"created_at,default:current_timestamp,notnull"`
	UpdatedAt      time.Time  `json:"updated_at"  bun:"updated_at,default:current_timestamp,notnull"`
	DeletedAt      *time.Time `json:"-,omitempty" bun:",soft_delete,nullzero"`
}

type FindOrganizationUserOptions struct {
	UserID         string    `json:"user_id"`
	Username       string    `json:"username"`
	OrganizationID uuid.UUID `json:"organization_id"`
}

type ListOrganizationUserOptions struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	Paginator      Paginator
}

type OrganizationUserRepository interface {
	Create(context.Context, *OrganizationUser) (*OrganizationUser, error)
	Find(context.Context, *FindOrganizationUserOptions) (*OrganizationUser, error)
	List(context.Context, *ListOrganizationUserOptions) ([]*OrganizationUser, int64, error)
}
