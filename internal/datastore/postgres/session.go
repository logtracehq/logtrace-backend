package postgres

import (
	"context"

	"github.com/uptrace/bun"
	"gitlab.com/logbase/logbase"
	"gitlab.com/logbase/logbase/internal/pkg/util"
)

type sessionRepo struct {
	inner *bun.DB
}

func NewSessionRepository(db *bun.DB) logbase.SessionRepository {
	return &sessionRepo{
		inner: db,
	}
}

func (s *sessionRepo) Create(ctx context.Context, session *logbase.Session) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := s.inner.NewInsert().Model(session).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (s *sessionRepo) List(ctx context.Context, opts *logbase.FindSessionOptions) (*logbase.Session, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	session := &logbase.Session{}

	sel := s.inner.NewSelect().Model(session)

	if !util.IsStringEmpty(opts.ID.String()) {
		sel = sel.Where("id = ?", opts.ID.String())
	}

	if !util.IsStringEmpty(opts.UserID.String()) {
		sel = sel.Where("user_id = ?", opts.UserID.String())
	}

	if !util.IsStringEmpty(opts.OrganizationID.String()) {
		sel = sel.Where("organization_id = ?", opts.OrganizationID.String())
	}

	err := sel.Scan(ctx)
	if err != nil {
		return session, err
	}
	return session, nil
}

func (s *sessionRepo) ListAll(ctx context.Context, opts *logbase.ListSessionsOptions) ([]*logbase.Session, int64, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	sessions := []*logbase.Session{}
	count := int64(0)

	totalCount, err := s.inner.NewSelect().Model(&sessions).Where("deleted_at IS NULL").Count(ctx)
	if err != nil {
		return nil, count, err
	}
	count = int64(totalCount)

	return sessions, count, s.inner.NewSelect().
		Model(&sessions).
		Limit(int(opts.Paginator.PerPage)).
		Offset(int(opts.Paginator.Offset())).
		Scan(ctx)
}
