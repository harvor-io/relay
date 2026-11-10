//go:build acceptance

package acceptance

import (
	"context"
	"crypto/rand"
	"database/sql"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"

	"github.com/harvor-io/relay/internal/bus"
	"github.com/harvor-io/relay/internal/config"
	"github.com/harvor-io/relay/internal/database"
	"github.com/harvor-io/relay/internal/database/sqlite"
	"github.com/harvor-io/relay/internal/handlers"
	"github.com/harvor-io/relay/internal/middleware"
	"github.com/harvor-io/relay/internal/models"
	"github.com/harvor-io/relay/internal/repositories"
	sqliterepo "github.com/harvor-io/relay/internal/repositories/sqlite"
	"github.com/harvor-io/relay/internal/services"
	"github.com/harvor-io/relay/pkg/encryptor/aes"
	secretssqlite "github.com/harvor-io/relay/pkg/secrets/sqlite"
)

// relay is Relay's source and key services over real SQLite repositories and
// secret store, in a database of one scenario's own, behind the same
// request-signing check and routes the app serves. Only the eventbus, a
// boundary these rules don't reach, is mocked.
type relay struct {
	dir       string
	db        *sql.DB
	sources   services.SourceService
	keys      services.SourceKeyService
	envelopes repositories.EnvelopeRepository
	router    http.Handler
}

// newRelay gives a scenario a freshly migrated database and Relay's services
// over it, wired as `relay serve` wires them.
func newRelay(ctx context.Context) (*relay, error) {
	dir, err := os.MkdirTemp("", "relay-acceptance-")
	if err != nil {
		return nil, err
	}
	r := &relay{dir: dir}

	r.db, err = database.Open(ctx, &config.Config{
		DatabaseDriver: sqlite.DriverName,
		DatabaseURL:    "file:" + filepath.Join(dir, "relay.db"),
	})
	if err != nil {
		return r, err
	}
	migrator, err := database.NewMigrator(r.db, sqlite.DriverName)
	if err != nil {
		return r, err
	}
	if _, err := migrator.Up(ctx); err != nil {
		return r, err
	}

	secretStore, err := secretssqlite.NewStore(ctx, r.db)
	if err != nil {
		return r, err
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return r, err
	}
	enc, err := aes.New(key)
	if err != nil {
		return r, err
	}

	eventBus := &bus.BusMock{
		PublishFunc: func(context.Context, *models.Envelope) error { return nil },
	}
	sourceRepo := sqliterepo.NewSourceRepository(r.db)
	keyRepo := sqliterepo.NewSourceKeyRepository(r.db)
	r.envelopes = sqliterepo.NewEnvelopeRepository(r.db)
	r.sources = services.NewSourceService(sourceRepo, r.envelopes, keyRepo, secretStore, eventBus)
	r.keys = services.NewSourceKeyService(keyRepo, sourceRepo, secretStore, enc)

	router := chi.NewRouter()
	handlers.NewSourceHandler(r.sources, middleware.HMACAuth(r.sources, r.keys)).RegisterRoutes(router)
	r.router = router
	return r, nil
}

// close closes the scenario's database and removes it.
func (r *relay) close() error {
	if r == nil {
		return nil
	}
	if r.db != nil {
		_ = r.db.Close()
	}
	return os.RemoveAll(r.dir)
}
