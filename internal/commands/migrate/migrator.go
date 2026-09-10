// Package migrate holds the subcommands of the `relay migrate` command tree.
// The parent command lives in the commands package one directory up.
package migrate

import (
	"context"
	"log/slog"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/harvor-io/relay/internal/config"
	"github.com/harvor-io/relay/internal/database"
	"github.com/harvor-io/relay/internal/logging"
)

// withMigrator opens the configured database, builds a Migrator, and passes it
// to fn, closing the connection afterwards. Command results are printed to
// stdout by fn; this wrapper logs the connection lifecycle to the structured
// logger on stderr.
func withMigrator(fn func(context.Context, *database.Migrator) error) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		cfg := config.New()

		logger := logging.New(os.Stderr, cfg.LogLevel, cfg.LogFormat)
		slog.SetDefault(logger)

		logger.Info("connecting to database",
			"command", cmd.Name,
			"database_driver", cfg.DatabaseDriver,
		)

		db, err := database.Open(ctx, cfg)
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		migrator, err := database.NewMigrator(db, cfg.DatabaseDriver)
		if err != nil {
			return err
		}

		if err := fn(ctx, migrator); err != nil {
			logger.Error("migration command failed", "command", cmd.Name, "error", err)
			return err
		}
		logger.Info("migration command complete", "command", cmd.Name)
		return nil
	}
}
