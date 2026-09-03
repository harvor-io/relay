package migrate

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/harvor-io/relay/internal/database"
)

// DownCommand rolls back the most recently applied migration.
func DownCommand() *cli.Command {
	return &cli.Command{
		Name:   "down",
		Usage:  "Roll back the most recently applied migration",
		Action: withMigrator(runDown),
	}
}

func runDown(ctx context.Context, m *database.Migrator) error {
	if err := m.Down(ctx); err != nil {
		return err
	}
	version, err := m.Version(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("rolled back one migration; schema is at version %d\n", version)
	return nil
}
