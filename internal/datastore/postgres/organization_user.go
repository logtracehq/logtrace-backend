package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"gitlab.com/logbase/logbase"
)

type organizationUserRepo struct {
	inner *bun.DB
}

func NewOrganizationUserRepository(db *bun.DB) logbase.OrganizationUserRepository {
	return &organizationUserRepo{
		inner: db,
	}
}

func (e *organizationUserRepo) Create(ctx context.Context, organizationUser *logbase.OrganizationUser) (*logbase.OrganizationUser, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := e.inner.NewInsert().Model(organizationUser).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return organizationUser, nil
}

func (e *organizationUserRepo) Find(ctx context.Context, opts *logbase.FindOrganizationUserOptions) (*logbase.OrganizationUser, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	organizationUser := &logbase.OrganizationUser{}

	query := e.inner.NewSelect().Model(organizationUser).Where("deleted_at IS NULL")

	if opts.UserID != "" {
		query = query.Where("user_id = ?", opts.UserID)
	}

	if opts.OrganizationID != uuid.Nil {
		query = query.Where("organization_id = ?", opts.OrganizationID)
	}

	if opts.Name != "" {
		query = query.Where("name = ?", opts.Name)
	}

	err := query.Scan(ctx)
	if err != nil {
		return nil, err
	}

	return organizationUser, nil
}

func (e *organizationUserRepo) List(ctx context.Context, opts *logbase.ListOrganizationUserOptions) ([]*logbase.OrganizationUser, int64, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	organizationUsers := []*logbase.OrganizationUser{}
	count := int64(0)

	query := e.inner.NewSelect().Model(organizationUsers).Where("deleted_at IS NULL")

	if opts.OrganizationID != uuid.Nil {
		query = query.Where("organization_id = ?", opts.OrganizationID)
	}

	totalCount, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	count = int64(totalCount)

	err = query.
		Offset(int(opts.Paginator.Offset())).
		Limit(int(opts.Paginator.PerPage)).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, 0, err
	}

	return organizationUsers, count, nil
}
