package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"gitlab.com/logbase/logbase"
)

type apiKeyRepo struct {
	inner *bun.DB
}

func NewAPIKeyRepository(db *bun.DB) logbase.APIKeyRepository {
	return &apiKeyRepo{
		inner: db,
	}
}

func (a *apiKeyRepo) FetchByValue(ctx context.Context, val string) (
	*logbase.APIKey, error,
) {
	apiKey := new(logbase.APIKey)

	err := a.inner.NewSelect().
		Model(apiKey).
		Where("value = ?", val).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		err = logbase.ErrAPIKeyNotFound
	}

	return apiKey, err
}

func (a *apiKeyRepo) Fetch(ctx context.Context, opts logbase.APIKeyOptions) (
	*logbase.APIKey, error,
) {
	apiKey := new(logbase.APIKey)
	query := a.inner.NewSelect().Model(apiKey)

	if opts.ID != uuid.Nil {
		query = query.Where("id = ?", opts.ID)
	}
	if opts.UserID != uuid.Nil {
		query = query.Where("user_id = ?", opts.UserID)
	}
	if opts.OrganizationID != uuid.Nil {
		query = query.Where("organization_id = ?", opts.OrganizationID)
	}

	err := query.Order("created_at DESC").Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		err = logbase.ErrAPIKeyNotFound
	}
	return apiKey, err
}

func (r *apiKeyRepo) List(ctx context.Context, opts logbase.APIKeyOptions) ([]*logbase.APIKey, error) {
	var apiKeys []*logbase.APIKey

	return apiKeys, r.inner.NewSelect().Model(&apiKeys).Relation("Resource").
		Where("organization_id = ?", opts.OrganizationID).
		Scan(ctx)
}

func (r *apiKeyRepo) Create(ctx context.Context, apiKey *logbase.APIKey) error {
	return r.inner.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().Model(apiKey).Exec(ctx)
		if err != nil {
			return err
		}

		count, err := tx.NewSelect().
			Model(new(logbase.APIKey)).
			Where("organization_id = ?", apiKey.OrganizationID).
			Count(ctx)
		if err != nil {
			return err
		}

		if count > 10 {
			return logbase.ErrAPIKeyMaxLimit
		}

		return nil
	})
}

func (r *apiKeyRepo) Revoke(ctx context.Context, opts logbase.APIKeyOptions) error {
	now := time.Now()

	q := r.inner.NewUpdate().
		Model(opts.APIKey).
		Set("updated_at = ?", now)

	switch opts.RevocationType {
	case logbase.RevocationTypeImmediate:
		q = q.Set("expires_at = CURRENT_DATE + INTERVAL '1 day' - INTERVAL '1 hour'")
		q = q.Set("deleted_at = NOW()")
	case logbase.RevocationTypeDay:
		q = q.Set("expires_at = CURRENT_DATE + INTERVAL '2 days' - INTERVAL '1 hour'")
	case logbase.RevocationTypeWeek:
		q = q.Set("expires_at = CURRENT_DATE + INTERVAL '8 days' - INTERVAL '1 hour'")
	}

	q = q.Where("id = ?", opts.APIKey.ID)

	_, err := q.Exec(ctx)
	return err
}

func (a *apiKeyRepo) FetchByName(ctx context.Context, name string, organizationID uuid.UUID) (*logbase.APIKey, error) {
	apiKey := new(logbase.APIKey)

	err := a.inner.NewSelect().
		Model(apiKey).
		Where("name = ? AND organization_id = ?", name, organizationID).
		Scan(ctx)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, logbase.ErrAPIKeyNotFound
	}

	return apiKey, err
}
