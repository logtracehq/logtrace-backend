## Logbase

Logbase is a secure audit trail and logging system designed to provide tamper-evident logging capabilities
for applications and systems.

### Features

- **Tamper-Evident Audit Logs**: Cryptographic hashing ensures log integrity and prevents unauthorized modifications
- **Multi-Tenant Architecture**: Organization-based access control with role-based permissions
- **RESTful API**: Comprehensive API for log management, user administration, and system monitoring
- **JWT Authentication**: Secure token-based authentication with optional OAuth integration
- **Real-Time Monitoring**: Live event streaming and alerting capabilities
- **Compliance Ready**: Built-in features for GDPR, SOC 2, and other regulatory requirements
- **PostgreSQL Backend**: Robust database storage with migration support
- **Web Dashboard**: Intuitive React-based interface for log visualization and management

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

Using the Taskfile targets (Go runner invokes `golang-migrate` internally):

```bash
task migrate                 # apply all up migrations
task migrate-down               # roll back last migration (1 step)
task migrate-force VERSION=2    # force set version
task migrate-create NAME=add_new_table  # create a new migration pair
```

Direct CLI example:

```bash
migrate -path ./migrations -database "postgres://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=$DB_SSLMODE" up
```

#### Creating New Migrations

```bash
task migrate-create --- descriptive_change
# This produces something like: 000002_descriptive_change.up.sql / .down.sql
```

Edit the generated files with your SQL changes, then run `task migrate`.

#### Forcing Version (Recovery)

If you need to manually set the schema version (e.g. after fixing a broken migration):

```bash
task migrate-force VERSION=1
```

#### Notes

- The initial migration enables the `uuid-ossp` extension.
- Tables use `timestamptz` and soft-delete columns (`deleted_at`) matching the Bun models.
- Keep application models and migrations in sync to avoid runtime issues.
- Foreign keys added in migration `000002_add_foreign_keys` for organizations (plan), sessions (organization), and audit logs (user).

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
