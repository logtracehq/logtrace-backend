package postgres

import (
	"context"
	"html"
	"strings"
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

func (s *sessionRepo) ListAll(
	ctx context.Context,
	opts *logbase.ListSessionsOptions,
) ([]*logbase.Session, int64, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	var sessions []*logbase.Session
	var count int64

	buildQuery := func(q *bun.SelectQuery) *bun.SelectQuery {
		if util.IsStringEmpty(opts.Status) {
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

	countQuery := buildQuery(
		s.inner.NewSelect().Model((*logbase.Session)(nil)),
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

func (s *sessionRepo) Metrics(
	ctx context.Context,
	opts *logbase.FindSessionOptions,
) (int64, int64, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	last24h := time.Now().Add(-24 * time.Hour)

	query := s.inner.NewSelect().
		Model((*logbase.Session)(nil)).
		Where("deleted_at IS NULL").
		Where("organization_id = ?", opts.OrganizationID.String())

	countQuery := query.Where("created_at >= ?", last24h)
	suspiciousCountQuery := query.Where("user_id = NULL AND username = NULL").Where("created_at >= ?", last24h)

	totalCount, err := countQuery.Count(ctx)
	suspiciousCount, _ := suspiciousCountQuery.Count(ctx)
	if err != nil {
		return 0, 0, err
	}

	return int64(totalCount), int64(suspiciousCount), nil
}
