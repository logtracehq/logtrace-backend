package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/terra-consults/logbase"
	"github.com/terra-consults/logbase/internal/pkg/util"
	"github.com/uptrace/bun"
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

func (r *userRepo) Get(ctx context.Context, opts *logbase.FindUserOptions) (*logbase.User, error) {
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

	err := selectUser.Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, logbase.ErrUserNotFound
		}

		return nil, err
	}

	return user, nil
}

func (r *userRepo) Update(ctx context.Context, u *logbase.User) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := r.inner.NewUpdate().Model(&u).Where("id = ?", u.ID).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
