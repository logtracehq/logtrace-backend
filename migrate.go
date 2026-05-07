package logtrace

import "embed"

//go:embed internal/datastore/postgres/migrations
var Migrations embed.FS
