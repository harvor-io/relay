// Package database opens the application's database connection from
// configuration and runs schema migrations. Engine-specific connection code
// lives in the sqlite and libsql sub-packages; this package chooses between
// them based on config.Config.DatabaseDriver.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/harvor-io/relay/internal/config"
	"github.com/harvor-io/relay/internal/database/libsql"
	"github.com/harvor-io/relay/internal/database/sqlite"
)

const pingTimeout = 5 * time.Second

// Open connects to the database described by cfg and verifies the connection
// with a ping. The caller owns closing the returned *sql.DB.
func Open(ctx context.Context, cfg *config.Config) (*sql.DB, error) {
	var (
		db  *sql.DB
		err error
	)

	switch cfg.DatabaseDriver {
	case sqlite.DriverName:
		db, err = sqlite.Open(cfg.DatabaseURL)
	case libsql.DriverName:
		db, err = libsql.Open(cfg.DatabaseURL, cfg.DatabaseAuthToken)
	default:
		return nil, fmt.Errorf("database: unknown driver %q (want %q or %q)",
			cfg.DatabaseDriver, sqlite.DriverName, libsql.DriverName)
	}
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("database: ping %q: %w", cfg.DatabaseDriver, err)
	}

	return db, nil
}
