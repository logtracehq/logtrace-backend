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

type eventRepo struct {
	inner *bun.DB
}

func NewEventRepository(db *bun.DB) logtrace.EventRepository {
	return &eventRepo{
		inner: db,
	}
}

func (e *eventRepo) Create(ctx context.Context, event *logtrace.Event) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := e.inner.NewInsert().Model(event).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (e *eventRepo) List(ctx context.Context, opts logtrace.ListEventOptions) (*logtrace.Event, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	event := &logtrace.Event{}

	sel := e.inner.NewSelect().Model(event).Where("organization_id = ?", opts.OrganizationID)

	if !util.IsStringEmpty(opts.ID.String()) {
		sel = sel.Where("id = ?", opts.ID.String())
	}

	if !util.IsStringEmpty(opts.Action) {
		sel = sel.Where("action_name = ?", opts.Action)
	}

	err := sel.Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return event, logtrace.ErrEventNotFound
	}
	return event, err
}

func (e *eventRepo) ListAll(ctx context.Context, opts *logtrace.ListEventOptions) ([]*logtrace.Event, int64, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	var events []*logtrace.Event
	var count int64

	buildQuery := func(q *bun.SelectQuery) *bun.SelectQuery {
		if !util.IsStringEmpty(opts.OrganizationID.String()) {
			q = q.Where("organization_id = ?", opts.OrganizationID)
		}

		if !util.IsStringEmpty(opts.HTTPStatus) {
			if len(opts.HTTPStatus) == 3 && strings.HasSuffix(opts.HTTPStatus, "xx") {
				prefix := string(opts.HTTPStatus[0])
				q = q.Where("http_status LIKE ?", prefix+"%")
			} else {
				q = q.Where("http_status = ?", opts.HTTPStatus)
			}
		}

		if !util.IsStringEmpty(opts.HTTPMethod) {
			q = q.Where("http_method = ?", opts.HTTPMethod)
		}

		if !util.IsStringEmpty(opts.Action) {
			q = q.Where("action_name = ?", opts.Action)
		}

		if !util.IsStringEmpty(opts.UserID) {
			q = q.Where("user_id = ?", opts.UserID)
		}

		if !util.IsStringEmpty(opts.Username) {
			q = q.Where("username = ?", opts.Username)
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
				return q.WhereOr("user_id::text ILIKE ?", searchTerm).
					WhereOr("http_status ILIKE ?", searchTerm).
					WhereOr("http_method ILIKE ?", searchTerm).
					WhereOr("action_name ILIKE ?", searchTerm).
					WhereOr("username ILIKE ?", searchTerm).
					WhereOr("client_user_agent ILIKE ?", searchTerm).
					WhereOr("geo_ip_location ILIKE ?", searchTerm).
					WhereOr("http_endpoint ILIKE ?", searchTerm)
			})
		}

		return q
	}

	countQuery := buildQuery(
		e.inner.NewSelect().Model((*logtrace.Event)(nil)),
	)

	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	count = int64(total)

	listQuery := buildQuery(e.inner.NewSelect().Model(&events))

	err = listQuery.
		Order("created_at DESC").
		Offset(int(opts.Paginator.Offset())).
		Limit(int(opts.Paginator.PerPage)).
		Scan(ctx)
	if err != nil {
		return nil, 0, err
	}

	return events, count, nil
}

func (e eventRepo) Metrics(
	ctx context.Context,
	opts *logtrace.ListEventOptions,
) (int64, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	var count int64

	last24h := time.Now().Add(-24 * time.Hour)

	countQuery := e.inner.NewSelect().
		Model((*logtrace.Event)(nil)).
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
