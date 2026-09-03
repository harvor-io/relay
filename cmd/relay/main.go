// Command relay is the entrypoint for the Relay event routing service.
package main

import (
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/harvor-io/relay/internal/commands"
)

// version is stamped at build time:
//
//	go build -ldflags "-X main.version=$(git describe --tags)" ./cmd/relay
var version = "dev"

func main() {
	cmd := &cli.Command{
		Name:        "relay",
		Usage:       "Relay event routing service",
		Description: "Open-source event ingestion and routing",
		Version:     version,
		Commands: []*cli.Command{
			commands.ServeCommand(),
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
