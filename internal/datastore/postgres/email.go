package postgres

import (
	"context"

	"github.com/uptrace/bun"
	"gitlab.com/logtrace/logtrace"
)

type emailRepo struct {
	inner *bun.DB
}

func NewEmailRepository(db *bun.DB) logtrace.EmailVerificationRepository {
	return &emailRepo{
		inner: db,
	}
}

func (ev *emailRepo) Create(ctx context.Context, emailVerification *logtrace.EmailVerification) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := ev.inner.NewInsert().Model(emailVerification).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
