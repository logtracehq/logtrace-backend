## Logbase

Logbase is an open source audit trail and logging system designed to provide secure, tamper-evident logging capabilities
for applications and systems.

### Features

_Coming soon..._

### Database Migrations

Logbase uses the `golang-migrate` tool for schema migrations.

#### Install CLI

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Ensure your `$GOBIN` (often `$GOPATH/bin`) is on your `PATH` so the `migrate` command is available.

#### Configuration

Database connection info is sourced via `viper` defaults defined in `config/config.go`. You can override any value by exporting an environment variable matching the key with dots replaced by underscores, e.g. `export DATABASE_DB_PASSWORD=secret` overrides `database.db_password`.

#### Run Migrations

Using the Makefile targets (Go runner invokes `golang-migrate` internally):

```bash
make migrate-up                 # apply all up migrations
make migrate-down               # roll back last migration (1 step)
make migrate-force VERSION=2    # force set version
make migrate-create NAME=add_new_table  # create a new migration pair
```

Direct CLI example:

```bash
migrate -path ./migrations -database "postgres://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=$DB_SSLMODE" up
```

#### Creating New Migrations

```bash
make migrate-create NAME=descriptive_change
# This produces something like: 000002_descriptive_change.up.sql / .down.sql
```

Edit the generated files with your SQL changes, then run `make migrate-up`.

#### Forcing Version (Recovery)

If you need to manually set the schema version (e.g. after fixing a broken migration):

```bash
make migrate-force VERSION=1
```

#### Notes

- The initial migration enables the `uuid-ossp` extension.
- Tables use `timestamptz` and soft-delete columns (`deleted_at`) matching the Bun models.
- Keep application models and migrations in sync to avoid runtime issues.
- Foreign keys added in migration `000002_add_foreign_keys` for organizations (plan), sessions (organization), and audit logs (resource, user). Audit log `resource_id` and `user_id` columns are now UUIDs.

### Go Migration Runner

You can also run migrations programmatically:

```bash
go run cmd/migrate/main.go -action=up
go run cmd/migrate/main.go -action=down -steps=1
go run cmd/migrate/main.go -action=force -version=2
go run cmd/migrate/main.go -action=version
```

Flags:

- `-path`: path to migrations (default `./migrations`)
- `-action`: `up|down|force|version`
- `-steps`: number of steps for `down` (default 1)
- `-version`: version number for `force`

No explicit `DB_*` environment variables required; viper provides defaults unless you override them.
