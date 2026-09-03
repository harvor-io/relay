// Package migrate holds the subcommands of the `relay migrate` command tree.
// The parent command lives in the commands package one directory up.
package migrate

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/harvor-io/relay/internal/config"
	"github.com/harvor-io/relay/internal/database"
)

// withMigrator opens the configured database, builds a Migrator, and passes it
// to fn, closing the connection afterwards.
func withMigrator(fn func(context.Context, *database.Migrator) error) cli.ActionFunc {
	return func(ctx context.Context, _ *cli.Command) error {
		cfg := config.New()

		db, err := database.Open(ctx, cfg)
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		migrator, err := database.NewMigrator(db, cfg.DatabaseDriver)
		if err != nil {
			return err
		}
		return fn(ctx, migrator)
	}
}
