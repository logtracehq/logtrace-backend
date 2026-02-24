package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"gitlab.com/logbase/logbase"
	"gitlab.com/logbase/logbase/internal/pkg/util"
)

type userRepo struct {
	inner *bun.DB
}

func NewUserRepository(db *bun.DB) logbase.UserRepository {
	return &userRepo{
		inner: db,
	}
}

func (r *userRepo) Create(ctx context.Context, u *logbase.User) (*logbase.User, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := r.inner.NewInsert().Model(u).Returning("*").Exec(ctx)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (r *userRepo) ListAll(ctx context.Context, opts *logbase.FindUserOptions) ([]*logbase.User, int64, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	var users []*logbase.User

	countSelect := r.inner.NewSelect().Model(&logbase.User{}).Where("deleted_at IS NULL")

	if !util.IsStringEmpty(opts.Email.String()) {
		countSelect = countSelect.Where("email = ?", opts.Email)
	}
	if opts.OrganizationID != uuid.Nil {
		countSelect = countSelect.Where("metadata->'organization_id' @> ?::jsonb", fmt.Sprintf("[\"%s\"]", opts.OrganizationID.String()))
	}

	selectUser := r.inner.NewSelect().Model(&users).Relation("Roles")
	if !util.IsStringEmpty(opts.Email.String()) {
		selectUser = selectUser.Where("email = ?", opts.Email)
	}
	if opts.OrganizationID != uuid.Nil {
		selectUser = selectUser.Where("metadata->'organization_id' @> ?::jsonb", fmt.Sprintf("[\"%s\"]", opts.OrganizationID.String()))
	}

	totalCount, err := countSelect.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	err = selectUser.Scan(ctx)
	if err != nil {
		return nil, 0, err
	}

	return users, int64(totalCount), nil
}

func (r *userRepo) List(ctx context.Context, opts *logbase.FindUserOptions) (*logbase.User, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	user := &logbase.User{
		Roles: []logbase.UserRole{},
	}

	selectUser := r.inner.NewSelect().Model(user).Relation("Roles")
	if !util.IsStringEmpty(opts.Email.String()) {
		selectUser = selectUser.Where("email = ?", opts.Email)
	}
	if opts.ID != uuid.Nil {
		selectUser = selectUser.Where("id = ?", opts.ID)
	}
	if opts.OrganizationID != uuid.Nil {
		selectUser = selectUser.Where("metadata->'organization_id' @> ?::jsonb", fmt.Sprintf("[\"%s\"]", opts.OrganizationID.String()))
	}

	err := selectUser.Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, logbase.ErrUserNotFound
		}

		return nil, err
	}

	return user, nil
}

func (r *userRepo) Update(ctx context.Context, u *logbase.User) (*logbase.User, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := r.inner.NewUpdate().Model(u).Where("id = ?", u.ID).
		OmitZero().Returning("*").Exec(ctx)
	if err != nil {
		return nil, err
	}
	return u, nil
}
