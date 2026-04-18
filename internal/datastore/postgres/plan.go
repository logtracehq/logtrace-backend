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

	var plan logtrace.Plan
	err := p.inner.NewSelect().Model(&plan).Where("id = ?", id).Scan(ctx)

	return &plan, err
}

func (p *planRepo) ListAll(ctx context.Context) ([]logtrace.Plan, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	var plans []logtrace.Plan
	err := p.inner.NewSelect().Model(&plans).Order("created_at DESC").Scan(ctx)
	if err != nil {
		return nil, err
	}

	return plans, nil
}

func (p *planRepo) Update(ctx context.Context, plan *logtrace.Plan) (*logtrace.Plan, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := p.inner.NewUpdate().Model(plan).Where("id = ?", plan.ID).OmitZero().
		Returning("*").Exec(ctx)
	if err != nil {
		return nil, err
	}

	return plan, nil
}

func (p *planRepo) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	var plan logtrace.Plan
	_, err := p.inner.NewDelete().Model(&plan).Where("id = ?", id).Exec(ctx)
	return err
}
