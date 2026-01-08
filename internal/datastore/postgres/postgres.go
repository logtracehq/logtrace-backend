package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/alexlast/bunzap"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bunotel"
	"gitlab.com/logbase/logbase/config"
	"go.uber.org/zap"
)

type Connection struct {
	DB *bun.DB
}

var timeout time.Duration

func withContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}
func New(cfg *config.Config, logger *zap.Logger) (*bun.DB, error) {
	pgdb := sql.OpenDB(
		pgdriver.NewConnector(
			pgdriver.WithDSN(
				cfg.Database.Postgres.DSN)))

	db := bun.NewDB(pgdb, pgdialect.New())

	if cfg.Database.Postgres.LogQueries {
		db.AddQueryHook(
			bunzap.NewQueryHook(
				bunzap.QueryHookOptions{
					Logger: logger,
				}))
	}

	db.AddQueryHook(
		bunotel.NewQueryHook(
			bunotel.WithDBName("logbase.database")))

	timeout = cfg.Database.Postgres.QueryTimeout
	return db, db.Ping()
}
