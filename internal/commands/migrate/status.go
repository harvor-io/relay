package migrate

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/harvor-io/relay/internal/database"
)

// StatusCommand shows which migrations have been applied.
func StatusCommand() *cli.Command {
	return &cli.Command{
		Name:   "status",
		Usage:  "Show which migrations have been applied",
		Action: withMigrator(runStatus),
	}
}

func runStatus(ctx context.Context, m *database.Migrator) error {
	statuses, err := m.Status(ctx)
	if err != nil {
		return err
	}
	for _, s := range statuses {
		state := "pending"
		if s.Applied {
			state = "applied"
		}
		fmt.Printf("%-8s %s\n", state, s.Name)
	}
	return nil
}
