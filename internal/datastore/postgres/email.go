package postgres

import (
	"context"

	"github.com/uptrace/bun"
	"gitlab.com/logbase/logbase"
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
