// Package commands contains the CLI subcommands.
package commands

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/urfave/cli/v3"

	"github.com/harvor-io/relay/internal/config"
	"github.com/harvor-io/relay/internal/database"
	"github.com/harvor-io/relay/internal/handlers"
	"github.com/harvor-io/relay/internal/middleware"
)

// ServeCommand starts the HTTP API.
func ServeCommand() *cli.Command {
	return &cli.Command{
		Name:        "serve",
		Usage:       "Start the HTTP API server",
		Description: "Serve the Relay HTTP API",
		Action: func(ctx context.Context, cmd *cli.Command) error {
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
			applied, err := migrator.Up(ctx)
			if err != nil {
				return err
			}
			if applied > 0 {
				log.Printf("serve: applied %d migration(s)", applied)
			}

			r := chi.NewRouter()
			r.Use(middleware.RequestID)
			r.Use(chimiddleware.Recoverer)

			r.Route("/api/v1", func(api chi.Router) {
				// Health stays public so infrastructure probes (Docker,
				// load balancers) can reach it without a token.
				handlers.NewHealthHandler().RegisterRoutes(api)
			})

			docsHandler, err := handlers.NewDocsHandler()
			if err != nil {
				return err
			}
			docsHandler.RegisterRoutes(r)

			srv := &http.Server{
				Addr:    ":" + cfg.Port,
				Handler: r,
			}

			log.Printf("serve: listening on %s", srv.Addr)
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return err
			}
			return nil
		},
	}
}
