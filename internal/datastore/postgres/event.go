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

func (e *eventRepo) Create(ctx context.Context, event logbase.Event) error {
	ctx, cancel := withContext(ctx)
	defer cancel()

	_, err := e.inner.NewInsert().Model(&event).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (e *eventRepo) List(ctx context.Context, opts logbase.EventFindOptions) (logbase.Event, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	event := logbase.Event{}

	sel := e.inner.NewSelect().Model(&event)

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

func (e *eventRepo) ListAll(ctx context.Context, page int, limit int) ([]logbase.Event, error) {
	ctx, cancel := withContext(ctx)
	defer cancel()

	events := []logbase.Event{}

	// TODO: Get the offset and limit from the URL params
	sel := e.inner.NewSelect().Model(&events).Offset(page).Limit(limit)
	err := sel.Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return events, logbase.ErrEventsNotFound
	}
	return events, err
}
