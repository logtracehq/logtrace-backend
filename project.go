package logbase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

var (
	ErrProjectNotFound     = LogbaseError("project not found")
	ErrProjectExists       = LogbaseError("project already exists")
	ErrProjectNameRequired = LogbaseError("project name is required")
	ErrProjectTypeRequired = LogbaseError("project type is required")
)

type Project struct {
	ID             uuid.UUID  `json:"id"          bun:"type:uuid,default:uuid_generate_v4(),pk"`
	Name           string     `json:"name" bun:"type:text,notnull"`
	Type           string     `json:"type" bun:"type:text,notnull"`
	OrganizationID uuid.UUID  `json:"organization_id" bun:"type:uuid,notnull"`
	CreatedAt      time.Time  `json:"created_at"  bun:"default:current_timestamp,notnull"`
	UpdatedAt      time.Time  `json:"updated_at"  bun:"default:current_timestamp,notnull"`
	DeletedAt      *time.Time `json:"-,omitempty" bun:",soft_delete,nullzero"`
	bun.BaseModel  `json:"-" bun:"table:projects"`
}

type ListProjectOptions struct {
	Paginator Paginator `json:"paginator"`
}

type FindProjectOptions struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type ProjectRepository interface {
	List(context.Context, uuid.UUID) (*Project, error)
	Create(context.Context, *Project) error
	ListAll(context.Context, ListProjectOptions) ([]*Project, int64, error)
	Delete(context.Context, uuid.UUID) error
}
