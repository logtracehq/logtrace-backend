package postgres

import (
	"context"

	"github.com/terra-consults/logbase"
	"github.com/uptrace/bun"
)

type emailRepo struct {
	inner *bun.DB
}

func NewEmailRepository(db *bun.DB) logbase.EmailVerificationRepository {
	return &emailRepo{
		inner: db,
	}
}

func (ev *emailRepo) Create(ctx context.Context, emailVerification *logbase.EmailVerification) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := ev.inner.NewInsert().Model(emailVerification).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
