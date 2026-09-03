package commands

import (
	"github.com/urfave/cli/v3"

	"github.com/harvor-io/relay/internal/commands/migrate"
)

// MigrateCommand manages the database schema.
func MigrateCommand() *cli.Command {
	return &cli.Command{
		Name:        "migrate",
		Usage:       "Manage database schema migrations",
		Description: "Apply, roll back, and inspect schema migrations",
		Commands: []*cli.Command{
			migrate.UpCommand(),
			migrate.DownCommand(),
			migrate.StatusCommand(),
		},
	}
}
