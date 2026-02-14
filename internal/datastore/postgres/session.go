package postgres

import (
	"context"
	"time"

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

	countSelect := s.inner.NewSelect().Model(&logbase.Session{}).Where("deleted_at IS NULL")
	if opts.Status != "" && opts.Status != "all" {
		countSelect = countSelect.Where("status = ?", opts.Status)
	}

	if !util.IsStringEmpty(opts.StartDate) {
		countSelect = countSelect.Where("DATE(login_at) >= ?", opts.StartDate)
	}

	if !util.IsStringEmpty(opts.EndDate) {
		countSelect = countSelect.Where("DATE(login_at) <= ?", opts.EndDate)
	}

	totalCount, err := countSelect.Count(ctx)
	if err != nil {
		return nil, count, err
	}
	count = int64(totalCount)

	listSelect := s.inner.NewSelect().Model(&sessions).Where("deleted_at IS NULL")

	if opts.Status != "" && opts.Status != "all" {
		listSelect = listSelect.Where("status = ?", opts.Status)
	}

	if !util.IsStringEmpty(opts.StartDate) {
		listSelect = listSelect.Where("DATE(login_at) >= ?", opts.StartDate)
	}

	if !util.IsStringEmpty(opts.EndDate) {
		listSelect = listSelect.Where("DATE(login_at) <= ?", opts.EndDate)
	}

	return sessions, count, listSelect.
		Limit(int(opts.Paginator.PerPage)).
		Offset(int(opts.Paginator.Offset())).
		Scan(ctx)
}

func (s *sessionRepo) Metrics(
	ctx context.Context,
	opts *logbase.FindSessionOptions,
) (int64, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	var count int64

	last24h := time.Now().Add(-24 * time.Hour)

	countQuery := s.inner.NewSelect().
		Model((*logbase.Session)(nil)).
		Where("deleted_at IS NULL").
		Where("login_at >= ?", last24h).
		Where("organization_id = ?", opts.OrganizationID.String())

	totalCount, err := countQuery.Count(ctx)
	if err != nil {
		return 0, err
	}

	count = int64(totalCount)

	return count, nil
}
