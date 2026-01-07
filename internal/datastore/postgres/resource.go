package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/terra-consults/logbase"
	"github.com/uptrace/bun"
)

type resourceRepo struct {
	inner *bun.DB
}

func NewResourceRepo(db *bun.DB) logbase.ResourceRepository {
	return &resourceRepo{
		inner: db,
	}
}

func (res *resourceRepo) Create(ctx context.Context, resource *logbase.Resource) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := res.inner.NewInsert().
		Model(resource).
		Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (res *resourceRepo) Get(ctx context.Context, id uuid.UUID) (*logbase.Resource, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	resource := &logbase.Resource{}
	err := res.inner.NewSelect().
		Model(resource).
		Where("id = ?", id).
		Scan(ctx)

	return resource, err
}

func (res *resourceRepo) ListAll(ctx context.Context, opts logbase.ListResourceOptions) ([]*logbase.Resource, int64, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	resources := []*logbase.Resource{}
	count := int64(0)
	countTotal, err := res.inner.NewSelect().Model(&(resources)).Where("deleted_at IS NULL").Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	count = int64(countTotal)

	return resources, count, res.inner.NewSelect().
		Model(&resources).
		Where("deleted_at IS NULL").
		Limit(int(opts.Paginator.PerPage)).
		Offset(int(opts.Paginator.Offset())).Scan(ctx)
}

func (res *resourceRepo) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := res.inner.NewDelete().
		Model(&logbase.Resource{}).
		Where("id = ?", id).
		Exec(ctx)
	return err
}
