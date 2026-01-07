package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/terra-consults/logbase"
	"github.com/uptrace/bun"
)

type passwordRepo struct {
	inner *bun.DB
}

func NewPasswordRepository(db *bun.DB) logbase.PasswordRepository {
	return &passwordRepo{
		inner: db,
	}
}

func (p *passwordRepo) Create(ctx context.Context, password *logbase.Password) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := p.inner.NewInsert().Model(password).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (p *passwordRepo) Get(ctx context.Context, userID uuid.UUID) (*logbase.Password, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	password := &logbase.Password{}

	err := p.inner.NewSelect().Model(password).Where("user_id = ?", userID).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, logbase.ErrPasswordNotFound
	}

	return password, err
}
