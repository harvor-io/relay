// Package commands contains the CLI subcommands.
package commands

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/urfave/cli/v3"

	"github.com/harvor-io/relay/internal/config"
	"github.com/harvor-io/relay/internal/database"
	"github.com/harvor-io/relay/internal/handlers"
	"github.com/harvor-io/relay/internal/logging"
	"github.com/harvor-io/relay/internal/middleware"
	sqliterepo "github.com/harvor-io/relay/internal/repositories/sqlite"
	"github.com/harvor-io/relay/internal/services"
	"github.com/harvor-io/relay/pkg/encryptor/aes"
	secretssqlite "github.com/harvor-io/relay/pkg/secrets/sqlite"
)

// shutdownTimeout bounds how long in-flight requests have to finish once a
// shutdown signal arrives before the server stops waiting.
const shutdownTimeout = 15 * time.Second

// ServeCommand starts the HTTP API.
func ServeCommand() *cli.Command {
	return &cli.Command{
		Name:        "serve",
		Usage:       "Start the HTTP API server",
		Description: "Serve the Relay HTTP API",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg := config.New()

			logger := logging.New(os.Stderr, cfg.LogLevel, cfg.LogFormat)
			slog.SetDefault(logger)

			logger.Info("starting relay",
				"port", cfg.Port,
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
			applied, err := migrator.Up(ctx)
			if err != nil {
				return err
			}
			if applied > 0 {
				logger.Info("applied database migrations", "count", applied)
			}

			sourceRepo := sqliterepo.NewSourceRepository(db)
			sourceService := services.NewSourceService(sourceRepo)

			destinationRepo := sqliterepo.NewDestinationRepository(db)

			secretsDB, err := database.Open(ctx, &config.Config{
				DatabaseDriver:    cfg.Secrets.Driver,
				DatabaseURL:       cfg.Secrets.DatabaseURL,
				DatabaseAuthToken: cfg.DatabaseAuthToken,
			})
			if err != nil {
				return err
			}
			defer func() { _ = secretsDB.Close() }()

			secretStore, err := secretssqlite.NewStore(ctx, secretsDB)
			if err != nil {
				return err
			}

			encryptionKey, err := base64.StdEncoding.DecodeString(cfg.Secrets.EncryptionKey)
			if err != nil {
				return fmt.Errorf("invalid SECRETS_ENCRYPTION_KEY: %w", err)
			}
			enc, err := aes.New(encryptionKey)
			if err != nil {
				return fmt.Errorf("invalid SECRETS_ENCRYPTION_KEY: %w", err)
			}

			sourceKeyService := services.NewSourceKeyService(
				sqliterepo.NewSourceKeyRepository(db),
				sourceRepo,
				secretStore,
				enc,
			)

			destinationService := services.NewDestinationService(destinationRepo, secretStore, enc)

			r := chi.NewRouter()
			r.Use(middleware.RequestID)
			r.Use(middleware.RequestLogger(logger))
			r.Use(chimiddleware.Recoverer)

			r.Route("/api/v1", func(api chi.Router) {
				// Health stays public so infrastructure probes (Docker,
				// load balancers) can reach it without a token.
				handlers.NewHealthHandler().RegisterRoutes(api)
				handlers.NewSourceHandler(sourceService).RegisterRoutes(api)
				handlers.NewSourceKeyHandler(sourceKeyService).RegisterRoutes(api)
				handlers.NewDestinationHandler(destinationService).RegisterRoutes(api)
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

			// Trip ctx on the first interrupt/terminate signal so the select
			// below can start a graceful drain.
			ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
			defer stop()

			serveErr := make(chan error, 1)
			go func() {
				logger.Info("listening", "addr", srv.Addr)
				err := srv.ListenAndServe()
				if errors.Is(err, http.ErrServerClosed) {
					err = nil
				}
				serveErr <- err
			}()

			select {
			case err := <-serveErr:
				if err != nil {
					logger.Error("server stopped unexpectedly", "error", err)
				}
				return err
			case <-ctx.Done():
				logger.Info("shutdown signal received, draining connections",
					"timeout", shutdownTimeout,
				)
				shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
				defer cancel()
				if err := srv.Shutdown(shutdownCtx); err != nil {
					logger.Error("graceful shutdown failed", "error", err)
					return err
				}
				logger.Info("shutdown complete")
				return nil
			}
		},
	}
}
