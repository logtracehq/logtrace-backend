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

func (ev *emailRepo) Delete(ctx context.Context, token string) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := ev.inner.NewDelete().
		Model((*logtrace.EmailVerification)(nil)).
		Where("token = ?", token).
		Exec(ctx)

	return err
}

func (ev *emailRepo) List(ctx context.Context, token string) (*logtrace.EmailVerification, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	verification := new(logtrace.EmailVerification)
	err := ev.inner.NewSelect().Model(verification).Where("token = ?", token).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return verification, nil
}
