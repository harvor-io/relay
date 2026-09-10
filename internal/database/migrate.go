package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path"

	"github.com/pressly/goose/v3"

	"github.com/harvor-io/relay/internal/database/libsql"
	"github.com/harvor-io/relay/internal/database/migrations"
	"github.com/harvor-io/relay/internal/database/sqlite"
)

// Migrator applies and inspects the schema migrations embedded in the
// migrations package. It wraps a goose provider so the rest of the app never
// imports the migration library directly.
type Migrator struct {
	provider *goose.Provider
}

// MigrationStatus reports whether one migration file has been applied.
type MigrationStatus struct {
	Name    string
	Version int64
	Applied bool
}

// NewMigrator builds a Migrator for db. The migration set and SQL dialect are
// chosen from driver; sqlite and libsql share the "sqlite" migrations because
// they use the same dialect.
func NewMigrator(db *sql.DB, driver string) (*Migrator, error) {
	var dir string
	switch driver {
	case sqlite.DriverName, libsql.DriverName:
		dir = "sqlite"
	default:
		return nil, fmt.Errorf("database: no migrations for driver %q", driver)
	}

	sub, err := fs.Sub(migrations.FS, dir)
	if err != nil {
		return nil, fmt.Errorf("database: locate %q migrations: %w", dir, err)
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, db, sub)
	if err != nil {
		return nil, fmt.Errorf("database: init migrator: %w", err)
	}
	return &Migrator{provider: provider}, nil
}

// Up applies every pending migration and returns the number applied.
func (m *Migrator) Up(ctx context.Context) (int, error) {
	results, err := m.provider.Up(ctx)
	if err != nil {
		return 0, fmt.Errorf("database: migrate up: %w", err)
	}
	return len(results), nil
}

// Down rolls back the most recently applied migration.
func (m *Migrator) Down(ctx context.Context) error {
	if _, err := m.provider.Down(ctx); err != nil {
		return fmt.Errorf("database: migrate down: %w", err)
	}
	return nil
}

// Version returns the current schema version, or 0 when nothing is applied.
func (m *Migrator) Version(ctx context.Context) (int64, error) {
	v, err := m.provider.GetDBVersion(ctx)
	if err != nil {
		return 0, fmt.Errorf("database: schema version: %w", err)
	}
	return v, nil
}

// Status lists every migration file in order with its applied state.
func (m *Migrator) Status(ctx context.Context) ([]MigrationStatus, error) {
	raw, err := m.provider.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("database: migration status: %w", err)
	}

	out := make([]MigrationStatus, 0, len(raw))
	for _, s := range raw {
		out = append(out, MigrationStatus{
			Name:    path.Base(s.Source.Path),
			Version: s.Source.Version,
			Applied: s.State == goose.StateApplied,
		})
	}
	return out, nil
}
