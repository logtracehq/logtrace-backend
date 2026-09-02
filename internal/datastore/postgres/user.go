package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"gitlab.com/logtrace/logtrace"
	"gitlab.com/logtrace/logtrace/internal/pkg/util"
)

type userRepo struct {
	inner *bun.DB
}

func NewUserRepository(db *bun.DB) logtrace.UserRepository {
	return &userRepo{
		inner: db,
	}
}

func (r *userRepo) Create(ctx context.Context, u *logtrace.User) (*logtrace.User, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := r.inner.NewInsert().Model(u).Returning("*").Exec(ctx)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func organizationIDFilter(orgID uuid.UUID) (string, error) {
	if orgID == uuid.Nil {
		return "", nil
	}

	payload, err := json.Marshal([]uuid.UUID{orgID})
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func (r *userRepo) ListAll(ctx context.Context, opts *logtrace.FindUserOptions) ([]*logtrace.User, int64, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	var users []*logtrace.User

	countSelect := r.inner.NewSelect().Model(&logtrace.User{}).Where("deleted_at IS NULL")

	if !util.IsStringEmpty(opts.Email.String()) {
		countSelect = countSelect.Where("email = ?", opts.Email)
	}
	if opts.OrganizationID != uuid.Nil {
		orgIDJSON, err := organizationIDFilter(opts.OrganizationID)
		if err != nil {
			return nil, 0, err
		}
		countSelect = countSelect.Where("metadata->'organization_id' @> ?::jsonb", orgIDJSON)
	}

	selectUser := r.inner.NewSelect().Model(&users).Relation("Roles")
	if !util.IsStringEmpty(opts.Email.String()) {
		selectUser = selectUser.Where("email = ?", opts.Email)
	}
	if opts.OrganizationID != uuid.Nil {
		orgIDJSON, err := organizationIDFilter(opts.OrganizationID)
		if err != nil {
			return nil, 0, err
		}
		selectUser = selectUser.Where("metadata->'organization_id' @> ?::jsonb", orgIDJSON)
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

func (r *userRepo) List(ctx context.Context, opts *logtrace.FindUserOptions) (*logtrace.User, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	user := &logtrace.User{
		Roles: []logtrace.UserRole{},
	}

	selectUser := r.inner.NewSelect().Model(user).Relation("Roles")
	if !util.IsStringEmpty(opts.Email.String()) {
		selectUser = selectUser.Where("email = ?", opts.Email)
	}
	if opts.ID != uuid.Nil {
		selectUser = selectUser.Where("id = ?", opts.ID)
	}
	if opts.OrganizationID != uuid.Nil {
		orgIDJSON, err := organizationIDFilter(opts.OrganizationID)
		if err != nil {
			return nil, err
		}
		selectUser = selectUser.Where("metadata->'organization_id' @> ?::jsonb", orgIDJSON)
	}

	err := selectUser.Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, logtrace.ErrUserNotFound
		}

		return nil, err
	}

	return user, nil
}

func (r *userRepo) Update(ctx context.Context, u *logtrace.User) (*logtrace.User, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := r.inner.NewUpdate().Model(u).Where("id = ?", u.ID).
		OmitZero().Returning("*").Exec(ctx)
	if err != nil {
		return nil, err
	}
	return u, nil
}
