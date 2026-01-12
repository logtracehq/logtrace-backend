package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"gitlab.com/logbase/logbase"
	"gitlab.com/logbase/logbase/internal/pkg/util"
)

type orgRepo struct {
	inner *bun.DB
}

func NewOrganizationRepository(db *bun.DB) logbase.OrganizationRepository {
	return &orgRepo{
		inner: db,
	}
}

func (org *orgRepo) Create(ctx context.Context, organization *logbase.Organization) (*logbase.Organization, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	_, err := org.inner.NewInsert().Model(organization).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return organization, nil
}

func (org *orgRepo) Get(ctx context.Context, opts logbase.FindOrganizationOptions) (*logbase.Organization, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	organization := &logbase.Organization{}

	sel := org.inner.NewSelect().Model(organization)
	if !util.IsStringEmpty(opts.ID.String()) {
		sel = sel.Where("id = ?", opts.ID.String())
	}

	if !util.IsStringEmpty(opts.Name) {
		sel = sel.Where("name = ?", opts.Name)
	}

	err := sel.Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return organization, logbase.OrganizationNotFound
	}

	return organization, err
}

func (org *orgRepo) Update(ctx context.Context, id uuid.UUID) (*logbase.Organization, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	organization := &logbase.Organization{}

	_, err := org.inner.NewUpdate().Model(organization).Where("id = ?", id.String()).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return organization, nil
}

func (org *orgRepo) Delete(ctx context.Context, opts *logbase.FindOrganizationOptions) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	organization := &logbase.Organization{}

	sel := org.inner.NewDelete().Model(organization)
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

func (org *orgRepo) List(ctx context.Context, user *logbase.User) ([]logbase.Organization, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	orgs := make([]logbase.Organization, 0)

	if user.MetaData == nil {
		return orgs, nil
	}

	return orgs, org.inner.NewSelect().
		Model(&orgs).
		Where("id = ? AND is_active = ?", user.MetaData.OrganizationID, true).
		Order("created_at DESC").
		Scan(ctx)
}
