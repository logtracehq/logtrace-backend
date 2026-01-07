package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/terra-consults/logbase"
	"github.com/uptrace/bun"
)

type planRepo struct {
	inner *bun.DB
}

func NewPlanRepository(db *bun.DB) logbase.PlanRepository {
	return &planRepo{
		inner: db,
	}
}

func (p *planRepo) Create(ctx context.Context, plan *logbase.Plan) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := p.inner.NewInsert().Model(plan).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (p *planRepo) Get(ctx context.Context, id uuid.UUID) (*logbase.Plan, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	plan := &logbase.Plan{}
	err := p.inner.NewSelect().Model(plan).Where("id = ?", id).Scan(ctx)

	return plan, err
}

func (p *planRepo) ListAll(ctx context.Context) ([]logbase.Plan, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	plans := make([]logbase.Plan, 0)
	err := p.inner.NewSelect().Model(&plans).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return plans, nil
}
