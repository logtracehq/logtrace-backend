package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"gitlab.com/logtrace/logtrace"
)

type passwordRepo struct {
	inner *bun.DB
}

func NewPasswordRepository(db *bun.DB) logtrace.PasswordRepository {
	return &passwordRepo{
		inner: db,
	}
}

func (p *passwordRepo) Create(ctx context.Context, password *logtrace.Password) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := p.inner.NewInsert().Model(password).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (p *passwordRepo) List(ctx context.Context, userID uuid.UUID) (*logtrace.Password, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	password := &logtrace.Password{}

	err := p.inner.NewSelect().Model(password).Where("user_id = ?", userID).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, logtrace.ErrPasswordNotFound
	}

	return password, err
}

func (p *passwordRepo) Update(ctx context.Context, password *logtrace.Password) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := p.inner.NewUpdate().Model(password).Where("user_id = ?", password.UserID).Exec(ctx)
	if err != nil {
		return nil
	}

	return nil
}
