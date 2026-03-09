package postgres

import (
	"context"
	"database/sql"
	"errors"
	"html"
	"strings"
	"time"

	"github.com/uptrace/bun"
	"gitlab.com/logtrace/logtrace"
	"gitlab.com/logtrace/logtrace/internal/pkg/util"
)

type auditLogRepo struct {
	inner *bun.DB
}

func NewAuditLogRepository(db *bun.DB) logtrace.AuditLogRepository {
	return &auditLogRepo{
		inner: db,
	}
}

func (e *auditLogRepo) Create(ctx context.Context, auditLog *logtrace.AuditLog) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := e.inner.NewInsert().Model(auditLog).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (a *auditLogRepo) ListAll(ctx context.Context, opts logtrace.FindAuditLogOptions) ([]*logtrace.AuditLog, int64, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	var auditLogs []*logtrace.AuditLog
	var count int64

	buildQuery := func(q *bun.SelectQuery) *bun.SelectQuery {
		if !util.IsStringEmpty(opts.UserID) {
			q = q.Where("user_id = ?", opts.UserID)
		}

		if !util.IsStringEmpty(opts.UserName) {
			q = q.Where("username = ?", opts.UserName)
		}

		if !util.IsStringEmpty(opts.StartDate) &&
			!util.IsStringEmpty(opts.EndDate) {

			q = q.Where(
				"created_at BETWEEN ? AND ?",
				opts.StartDate+" 00:00:00",
				opts.EndDate+" 23:59:59",
			)
		} else if !util.IsStringEmpty(opts.StartDate) {
			q = q.Where(
				"created_at >= ?",
				opts.StartDate+" 00:00:00",
			)
		} else if !util.IsStringEmpty(opts.EndDate) {
			q = q.Where(
				"created_at <= ?",
				opts.EndDate+" 23:59:59",
			)
		}

		if !util.IsStringEmpty(opts.Search) {
			searchTerm := "%" + html.EscapeString(strings.TrimSpace(opts.Search)) + "%"

			q = q.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.
					WhereOr("user_id::text ILIKE ?", searchTerm).
					WhereOr("username ILIKE ?", searchTerm).
					WhereOr("action ILIKE ?", searchTerm).
					WhereOr("ip_address ILIKE ?", searchTerm)
			})
		}

		return q
	}

	countQuery := buildQuery(
		a.inner.NewSelect().Model((*logtrace.AuditLog)(nil)),
	)

	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	count = int64(total)

	listQuery := buildQuery(
		a.inner.NewSelect().Model(&auditLogs),
	)

	err = listQuery.
		Order("created_at DESC").
		Offset(int(opts.Paginator.Offset())).
		Limit(int(opts.Paginator.PerPage)).
		Scan(ctx)
	if err != nil {
		return nil, 0, err
	}

	return auditLogs, count, nil
}

func (a *auditLogRepo) List(ctx context.Context, opts *logtrace.FindAuditLogOptions) (*logtrace.AuditLog, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	auditLog := &logtrace.AuditLog{}

	sel := a.inner.NewSelect().Model(auditLog)

	if !util.IsStringEmpty(opts.ID.String()) {
		sel = sel.Where("?TableAlias.id = ?", opts.ID.String())
	}

	err := sel.Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return auditLog, logtrace.ErrAuditLogNotFound
	}
	return auditLog, err
}

func (a *auditLogRepo) Delete(ctx context.Context, opts logtrace.FindAuditLogOptions) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	auditLog := &logtrace.AuditLog{}

	sel := a.inner.NewSelect().Model(&auditLog)

	if !util.IsStringEmpty(opts.ID.String()) {
		sel = sel.Where("id = ?", opts.ID.String())
	}
	err := sel.Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return logtrace.ErrAuditLogNotFound
	}
	if err != nil {
		return err
	}

	_, err = a.inner.NewDelete().Model(&auditLog).Where("id = ?", auditLog.ID).Exec(ctx)
	return err
}

func (a *auditLogRepo) Metrics(
	ctx context.Context,
	opts *logtrace.FindAuditLogOptions,
) (int64, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	var count int64

	last24h := time.Now().Add(-24 * time.Hour)

	countQuery := a.inner.NewSelect().
		Model((*logtrace.AuditLog)(nil)).
		Where("deleted_at IS NULL").
		Where("created_at >= ?", last24h).
		Where("organization_id = ?", opts.OrganizationID.String())

	totalCount, err := countQuery.Count(ctx)
	if err != nil {
		return 0, err
	}

	count = int64(totalCount)

	return count, nil
}
