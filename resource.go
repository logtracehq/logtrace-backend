package logbase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

var (
	ErrResourceNotFound     = LogbaseError("resource not found")
	ErrResourceExists       = LogbaseError("resource already exists")
	ErrResourceNameRequired = LogbaseError("resource name is required")
	ErrResourceTypeRequired = LogbaseError("resource type is required")
)

type Resource struct {
	ID            uuid.UUID  `json:"id"          bun:"type:uuid,default:uuid_generate_v4(),pk"`
	Name          string     `json:"name"`
	Type          string     `json:"type"`
	CreatedAt     time.Time  `json:"created_at"  bun:"default:current_timestamp,notnull"`
	UpdatedAt     time.Time  `json:"updated_at"  bun:"default:current_timestamp,notnull"`
	DeletedAt     *time.Time `json:"-,omitempty" bun:",soft_delete,nullzero"`
	bun.BaseModel `json:"-" bun:"table:resources"`
}

type ListResourceOptions struct {
	Paginator Paginator `json:"paginator"`
}

type FindResourceOptions struct {
	Name string `json:"name"`
}

type ResourceRepository interface {
	Get(context.Context, uuid.UUID) (*Resource, error)
	Create(context.Context, *Resource) error
	ListAll(context.Context, ListResourceOptions) ([]*Resource, int64, error)
	Delete(context.Context, uuid.UUID) error
}
