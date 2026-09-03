package migrate

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/harvor-io/relay/internal/database"
)

// UpCommand applies all pending migrations.
func UpCommand() *cli.Command {
	return &cli.Command{
		Name:   "up",
		Usage:  "Apply all pending migrations",
		Action: withMigrator(runUp),
	}
}

func runUp(ctx context.Context, m *database.Migrator) error {
	n, err := m.Up(ctx)
	if err != nil {
		return err
	}
	version, err := m.Version(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("applied %d migration(s); schema is at version %d\n", n, version)
	return nil
}
