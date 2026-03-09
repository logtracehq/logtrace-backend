package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"gitlab.com/logtrace/logtrace"
)

type planRepo struct {
	inner *bun.DB
}

func NewPlanRepository(db *bun.DB) logtrace.PlanRepository {
	return &planRepo{
		inner: db,
	}
}

func (p *planRepo) Create(ctx context.Context, plan *logtrace.Plan) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := p.inner.NewInsert().Model(plan).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (p *planRepo) List(ctx context.Context, id uuid.UUID) (*logtrace.Plan, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	plan := &logtrace.Plan{}
	err := p.inner.NewSelect().Model(plan).Where("id = ?", id).Scan(ctx)

	return plan, err
}

func (p *planRepo) ListAll(ctx context.Context) ([]logtrace.Plan, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	plans := make([]logtrace.Plan, 0)
	err := p.inner.NewSelect().Model(&plans).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return plans, nil
}

func (p *planRepo) Update(ctx context.Context, plan *logtrace.Plan) (*logtrace.Plan, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := p.inner.NewUpdate().Model(plan).Where("id = ?", plan.ID).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return plan, nil
}

func (p *planRepo) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := p.inner.NewDelete().Model((*logtrace.Plan)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}
