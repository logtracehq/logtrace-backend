package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/uptrace/bun"
	"gitlab.com/logbase/logbase"
	"gitlab.com/logbase/logbase/internal/pkg/util"
)

type eventRepo struct {
	inner *bun.DB
}

func NewEventRepository(db *bun.DB) logbase.EventRepository {
	return &eventRepo{
		inner: db,
	}
}

func (e *eventRepo) Create(ctx context.Context, event *logbase.Event) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := e.inner.NewInsert().Model(event).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (e *eventRepo) List(ctx context.Context, opts logbase.ListEventOptions) (*logbase.Event, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	event := &logbase.Event{}

	sel := e.inner.NewSelect().Model(event).Where("organization_id = ?", opts.OrganizationID)

	if !util.IsStringEmpty(opts.ID.String()) {
		sel = sel.Where("id = ?", opts.ID.String())
	}

	if !util.IsStringEmpty(opts.Action) {
		sel = sel.Where("action_name = ?", opts.Action)
	}

	err := sel.Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return event, logbase.ErrEventNotFound
	}
	return event, err
}

func (e *eventRepo) ListAll(ctx context.Context, opts *logbase.ListEventOptions) ([]*logbase.Event, int64, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	events := []*logbase.Event{}
	count := int64(0)

	countSelect := e.inner.NewSelect().Model(&logbase.Event{}).Where("deleted_at IS NULL")

	if !util.IsStringEmpty(opts.OrganizationID.String()) {
		countSelect = countSelect.Where("organization_id = ?", opts.OrganizationID)
	}

	if !util.IsStringEmpty(opts.HTTPStatus) {
		// Support grouped status filters like "2xx", "4xx", "5xx" by matching prefix.
		if len(opts.HTTPStatus) == 3 && strings.HasSuffix(opts.HTTPStatus, "xx") {
			prefix := string(opts.HTTPStatus[0])
			countSelect = countSelect.Where("http_status LIKE ?", prefix+"%")
		} else {
			countSelect = countSelect.Where("http_status = ?", opts.HTTPStatus)
		}
	}

	if !util.IsStringEmpty(opts.HTTPMethod) {
		countSelect = countSelect.Where("http_method = ?", opts.HTTPMethod)
	}

	if !util.IsStringEmpty(opts.Action) {
		countSelect = countSelect.Where("action_name = ?", opts.Action)
	}

	if !util.IsStringEmpty(opts.StartDate) {
		countSelect = countSelect.Where("DATE(created_at) >= ?", opts.StartDate)
	}

	if !util.IsStringEmpty(opts.EndDate) {
		countSelect = countSelect.Where("DATE(created_at) <= ?", opts.EndDate)
	}

	totalCount, err := countSelect.Count(ctx)
	if err != nil {
		return events, count, err
	}

	count = int64(totalCount)

	listSelect := e.inner.NewSelect().
		Model(&events).
		Where("deleted_at IS NULL")

	if !util.IsStringEmpty(opts.OrganizationID.String()) {
		listSelect = listSelect.Where("organization_id = ?", opts.OrganizationID)
	}

	if !util.IsStringEmpty(opts.HTTPStatus) {
		if len(opts.HTTPStatus) == 3 && strings.HasSuffix(opts.HTTPStatus, "xx") {
			prefix := string(opts.HTTPStatus[0])
			listSelect = listSelect.Where("http_status LIKE ?", prefix+"%")
		} else {
			listSelect = listSelect.Where("http_status = ?", opts.HTTPStatus)
		}
	}

	if !util.IsStringEmpty(opts.HTTPMethod) {
		listSelect = listSelect.Where("http_method = ?", opts.HTTPMethod)
	}

	if !util.IsStringEmpty(opts.Action) {
		listSelect = listSelect.Where("action_name = ?", opts.Action)
	}

	if !util.IsStringEmpty(opts.StartDate) {
		listSelect = listSelect.Where("DATE(created_at) >= ?", opts.StartDate)
	}

	if !util.IsStringEmpty(opts.EndDate) {
		listSelect = listSelect.Where("DATE(created_at) <= ?", opts.EndDate)
	}
	if !util.IsStringEmpty(opts.EndDate) && !util.IsStringEmpty(opts.StartDate) {
		listSelect = listSelect.Where("DATE(created_at) BETWEEN ? AND ?", opts.StartDate, opts.EndDate)
	}
	if !util.IsStringEmpty(opts.UserID) {
		listSelect = listSelect.Where("user_id = ?", opts.UserID)
	}
	if !util.IsStringEmpty(opts.Username) {
		listSelect = listSelect.Where("username = ?", opts.Username)
	}

	return events, count, listSelect.
		Offset(int(opts.Paginator.Offset())).
		Limit(int(opts.Paginator.PerPage)).
		Order("created_at DESC").
		Scan(ctx)
}

func (e eventRepo) Metrics(
	ctx context.Context,
	opts *logbase.ListEventOptions,
) (int64, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	var count int64

	last24h := time.Now().Add(-24 * time.Hour)

	countQuery := e.inner.NewSelect().
		Model((*logbase.Event)(nil)).
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
