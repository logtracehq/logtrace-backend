package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/uptrace/bun"
	"gitlab.com/logbase/logbase"
	"gitlab.com/logbase/logbase/internal/pkg/util"
)

type auditLogRepo struct {
	inner *bun.DB
}

func NewAuditLogRepository(db *bun.DB) logbase.AuditLogRepository {
	return &auditLogRepo{
		inner: db,
	}
}

func (e *auditLogRepo) Create(ctx context.Context, auditLog *logbase.AuditLog) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := e.inner.NewInsert().Model(auditLog).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (a *auditLogRepo) ListAll(ctx context.Context, opts logbase.FindAuditLogOptions) ([]*logbase.AuditLog, int64, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	auditLogs := []*logbase.AuditLog{}
	count := int64(0)

	totalCount, err := a.inner.NewSelect().
		Model(&auditLogs).Where("deleted_at IS NULL").
		Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	count = int64(totalCount)

	return auditLogs, count, a.inner.NewSelect().
		Model(&auditLogs).
		Offset(int(opts.Paginator.Offset())).
		Limit(int(opts.Paginator.PerPage)).
		Order("created_at DESC").
		Scan(ctx)
}

func (a *auditLogRepo) List(ctx context.Context, opts *logbase.FindAuditLogOptions) (*logbase.AuditLog, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	auditLog := &logbase.AuditLog{}

	sel := a.inner.NewSelect().Model(auditLog)

	if !util.IsStringEmpty(opts.ID.String()) {
		sel = sel.Where("?TableAlias.id = ?", opts.ID.String())
	}

	err := sel.Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return auditLog, logbase.ErrAuditLogNotFound
	}
	return auditLog, err
}

func (a *auditLogRepo) Delete(ctx context.Context, opts logbase.FindAuditLogOptions) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	auditLog := &logbase.AuditLog{}

	sel := a.inner.NewSelect().Model(&auditLog)

	if !util.IsStringEmpty(opts.ID.String()) {
		sel = sel.Where("id = ?", opts.ID.String())
	}
	err := sel.Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return logbase.ErrAuditLogNotFound
	}
	if err != nil {
		return err
	}

	_, err = a.inner.NewDelete().Model(&auditLog).Where("id = ?", auditLog.ID).Exec(ctx)
	return err
}
