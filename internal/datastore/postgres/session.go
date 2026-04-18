package postgres

import (
	"context"
	"html"
	"strings"
	"time"

	"github.com/uptrace/bun"
	"gitlab.com/logtrace/logtrace"
	"gitlab.com/logtrace/logtrace/internal/pkg/util"
)

type sessionRepo struct {
	inner *bun.DB
}

func NewSessionRepository(db *bun.DB) logtrace.SessionRepository {
	return &sessionRepo{
		inner: db,
	}
}

func (s *sessionRepo) Create(ctx context.Context, session *logtrace.Session) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := s.inner.NewInsert().Model(session).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (s *sessionRepo) List(ctx context.Context, opts *logtrace.FindSessionOptions) (*logtrace.Session, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	var session logtrace.Session

	sel := s.inner.NewSelect().Model(&session)

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
		return &session, err
	}
	return &session, nil
}

func (s *sessionRepo) ListAll(ctx context.Context, opts *logtrace.ListSessionsOptions,
) ([]*logtrace.Session, int64, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	var sessions []*logtrace.Session
	var count int64

	buildQuery := func(q *bun.SelectQuery) *bun.SelectQuery {
		if !util.IsStringEmpty(opts.OrganizationID.String()) {
			q = q.Where("organization_id = ?", opts.OrganizationID.String())
		}

		if !util.IsStringEmpty(opts.Status) {
			q = q.Where("status = ?", opts.Status)
		}

		if !util.IsStringEmpty(opts.StartDate) &&
			!util.IsStringEmpty(opts.EndDate) {
			q = q.Where("DATE(login_at) BETWEEN ? AND ?", opts.StartDate, opts.EndDate)
		} else if !util.IsStringEmpty(opts.StartDate) {
			q = q.Where("DATE(login_at) >= ?", opts.StartDate)
		} else if !util.IsStringEmpty(opts.EndDate) {
			q = q.Where("DATE(login_at) <= ?", opts.EndDate)
		}

		if !util.IsStringEmpty(opts.Search) {
			searchTerm := "%" + html.EscapeString(strings.TrimSpace(opts.Search)) + "%"

			q = q.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.
					WhereOr("user_id::text ILIKE ?", searchTerm).
					WhereOr("ip_address::text ILIKE ?", searchTerm).
					WhereOr("username ILIKE ?", searchTerm).
					WhereOr("location ILIKE ?", searchTerm).
					WhereOr("device_info ILIKE ?", searchTerm)
			})
		}

		return q
	}

	var session logtrace.Session
	countQuery := buildQuery(
		s.inner.NewSelect().Model(&session),
	)

	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	count = int64(total)

	listQuery := buildQuery(
		s.inner.NewSelect().Model(&sessions),
	)

	err = listQuery.
		Order("login_at DESC").
		Limit(int(opts.Paginator.PerPage)).
		Offset(int(opts.Paginator.Offset())).
		Scan(ctx)
	if err != nil {
		return nil, 0, err
	}

	return sessions, count, nil
}

func (s *sessionRepo) Metrics(ctx context.Context, opts *logtrace.FindSessionOptions) (int64, int64, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	last24h := time.Now().Add(-24 * time.Hour)

	var session logtrace.Session
	totalCount, err := s.inner.NewSelect().
		Model(&session).
		Where("deleted_at IS NULL").
		Where("organization_id = ?", opts.OrganizationID.String()).
		Where("created_at >= ?", last24h).
		Count(ctx)

	suspiciousCount, _ := s.inner.NewSelect().
		Model(&session).
		Where("deleted_at IS NULL").
		Where("organization_id = ?", opts.OrganizationID.String()).
		Where("user_id IS NULL AND username IS NULL AND token IS NULL").
		Where("created_at >= ?", last24h).
		Count(ctx)
	if err != nil {
		return 0, 0, err
	}

	return int64(totalCount), int64(suspiciousCount), nil
}

func (s *sessionRepo) Logout(ctx context.Context, opts *logtrace.FindSessionOptions) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	var session logtrace.Session
	q := s.inner.NewUpdate().
		Model(&session).
		Set("logout_at = ?", time.Now()).
		Where("organization_id = ?", opts.OrganizationID.String())

	if !util.IsStringEmpty(opts.Token) {
		q = q.Where("token = ?", opts.Token)
	}

	if !util.IsStringEmpty(opts.ID.String()) {
		q = q.Where("id = ?", opts.ID.String())
	}

	_, err := q.Exec(ctx)
	return err
}
