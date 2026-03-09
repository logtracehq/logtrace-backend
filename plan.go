package logtrace

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Feature string

func (f Feature) String() string { return strings.ToLower(string(f)) }

type Plan struct {
	ID          uuid.UUID  `json:"id"          bun:"type:uuid,default:uuid_generate_v4(),pk"`
	Name        string     `json:"name"        bun:"name,unique"`
	Price       float64    `json:"price"       bun:"price"`
	Description string     `json:"description" bun:"description"`
	Features    []Feature  `json:"features"    bun:"type:jsonb"`
	Period      string     `json:"period"      bun:"period"`
	CTA         string     `json:"cta"         bun:"cta"`
	CreatedAt   time.Time  `json:"created_at"  bun:"default:current_timestamp,notnull"`
	UpdatedAt   time.Time  `json:"updated_at"  bun:"default:current_timestamp,notnull"`
	DeletedAt   *time.Time `json:"-,omitempty" bun:",soft_delete,nullzero"`
}

type PlanRepository interface {
	Create(context.Context, *Plan) error
	List(context.Context, uuid.UUID) (*Plan, error)
	ListAll(context.Context) ([]Plan, error)
	Update(context.Context, *Plan) (*Plan, error)
	Delete(context.Context, uuid.UUID) error
}
