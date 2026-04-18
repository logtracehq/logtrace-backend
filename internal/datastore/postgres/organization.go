package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/uptrace/bun"
	"gitlab.com/logtrace/logtrace"
	"gitlab.com/logtrace/logtrace/internal/pkg/util"
)

type orgRepo struct {
	inner *bun.DB
}

func NewOrganizationRepository(db *bun.DB) logtrace.OrganizationRepository {
	return &orgRepo{
		inner: db,
	}
}

func (org *orgRepo) Create(ctx context.Context, organization *logtrace.Organization) (*logtrace.Organization, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	_, err := org.inner.NewInsert().Model(organization).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return organization, nil
}

func (org *orgRepo) List(ctx context.Context, opts logtrace.FindOrganizationOptions) (*logtrace.Organization, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var organization logtrace.Organization

	sel := org.inner.NewSelect().Model(&organization)
	if !util.IsStringEmpty(opts.ID.String()) {
		sel = sel.Where("id = ?", opts.ID.String())
	}

	if !util.IsStringEmpty(opts.Name) {
		sel = sel.Where("name = ?", opts.Name)
	}

	err := sel.Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return &organization, logtrace.ErrOrganizationNotFound
	}

	return &organization, err
}

func (org *orgRepo) Update(ctx context.Context, o *logtrace.Organization) (*logtrace.Organization, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := org.inner.NewUpdate().
		Model(o).
		Where("id = ?", o.ID).
		OmitZero().
		Returning("*").Exec(ctx)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (org *orgRepo) Delete(ctx context.Context, opts *logtrace.FindOrganizationOptions) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var organization logtrace.Organization

	sel := org.inner.NewDelete().Model(&organization)
	if !util.IsStringEmpty(opts.ID.String()) {
		sel = sel.Where("id = ?", opts.ID.String())
	}

	if !util.IsStringEmpty(opts.Name) {
		sel = sel.Where("name = ?", opts.Name)
	}

	_, err := sel.Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (org *orgRepo) ListAll(ctx context.Context, opts *logtrace.FindOrganizationOptions) ([]logtrace.Organization, int64, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	count := int64(0)
	var organizations []logtrace.Organization

	countSelect := org.inner.NewSelect().Model(&organizations).Where("deleted_at IS NULL")
	if !util.IsStringEmpty(opts.UserID.String()) {
		countSelect = countSelect.Where("id IN (SELECT elem::uuid FROM users u, jsonb_array_elements_text(u.metadata->'organization_id') AS elem WHERE u.id = ?)", opts.UserID.String())
	}
	if !util.IsStringEmpty(opts.Plan) {
		countSelect = countSelect.Where("plan_name = ?", opts.Plan)
	}

	totalCount, err := countSelect.Count(ctx)
	if err != nil {
		return organizations, count, err
	}

	count = int64(totalCount)

	listSelect := org.inner.NewSelect().Model(&organizations).Where("deleted_at IS NULL")
	if !util.IsStringEmpty(opts.UserID.String()) {
		listSelect = listSelect.Where("id IN (SELECT elem::uuid FROM users u, jsonb_array_elements_text(u.metadata->'organization_id') AS elem WHERE u.id = ?)", opts.UserID.String())
	}
	if !util.IsStringEmpty(opts.Plan) {
		listSelect = listSelect.Where("plan_name = ?", opts.Plan)
	}

	return organizations, count, listSelect.
		Limit(int(opts.Paginator.PerPage)).
		Offset(int(opts.Paginator.Offset())).
		Scan(ctx)
}
