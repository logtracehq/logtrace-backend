package logbase

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Plan struct {
	ID        uuid.UUID  `json:"id"          bun:"type:uuid,default:uuid_generate_v4(),pk"`
	Name      string     `json:"name"        bun:"name,unique"`
	CreatedAt time.Time  `json:"created_at"  bun:"default:current_timestamp,notnull"`
	UpdatedAt time.Time  `json:"updated_at"  bun:"default:current_timestamp,notnull"`
	DeletedAt *time.Time `json:"-,omitempty" bun:",soft_delete,nullzero"`
}

type PlanRepository interface {
	Create(context.Context, *Plan) error
	List(context.Context, uuid.UUID) (*Plan, error)
	ListAll(context.Context) ([]Plan, error)
}
