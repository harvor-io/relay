// Package migrations embeds the SQL schema migration files, grouped into one
// sub-directory per database engine. SQLite and libSQL share a directory
// because they use the same SQL dialect.
package migrations

import "embed"

// FS holds every migration file. Consumers select an engine's migrations with
// fs.Sub(FS, "<engine>").
//
//go:embed sqlite/*.sql
var FS embed.FS
