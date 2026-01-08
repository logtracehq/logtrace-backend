package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/terra-consults/logbase"
	"github.com/terra-consults/logbase/internal/pkg/util"
	"github.com/uptrace/bun"
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

	totalCount, err := e.inner.NewSelect().Model(&logbase.Event{}).Where("deleted_at IS NULL").Count(ctx)
	if err != nil {
		return events, count, err
	}

	count = int64(totalCount)

	return events, count, e.inner.NewSelect().
		Model(&events).
		Offset(int(opts.Paginator.Offset())).
		Limit(int(opts.Paginator.PerPage)).
		Order("created_at DESC").
		Scan(ctx)
}
