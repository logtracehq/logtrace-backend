# LogTrace — Backend

LogTrace is a secure audit trail and logging platform built for compliance and security. It gives developers and organizations a simple API to capture, store, and query records of everything that happens in their systems — who did what, when, from where, and how.

This directory contains the Go backend service that powers the LogTrace API.

## Logging Primitives

### Event Tracking

Captures HTTP activity across your application — action names, HTTP methods, status codes, endpoints, client IPs, user agents, and geolocation. Useful for tracking every meaningful interaction a user or service has with your system.

### Session Logging

Records user login and logout events enriched with device information, IP address, and location. Sessions carry a status (`ACTIVE` / `INACTIVE`), making it easy to audit access history and detect anomalies.

### Audit Logs

Explicit, structured audit trail entries with an action name, timestamp, user identity, IP address, and arbitrary JSON metadata. Suitable for representing anything from a database query to a configuration change to a billing action.

All three are tied to an organization in a multi-tenant model, so each customer or team gets a fully isolated, searchable audit trail.

## Features

- **Multi-Tenant Architecture**: Organization-based access control with role-based permissions
- **RESTful API**: Comprehensive API for log ingestion, user administration, and system monitoring
- **JWT Authentication**: Secure token-based authentication with optional OAuth integration
- **Compliance Ready**: Built for SOC 2, GDPR, HIPAA, and similar regulatory frameworks
- **PostgreSQL Backend**: Robust database storage with migration support via `golang-migrate`
- **SDK Support**: Official SDKs for Go, Python, TypeScript, and PHP

## Database Migrations

Logtrace uses the `golang-migrate` tool for schema migrations.

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

## Go Migration Runner

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
