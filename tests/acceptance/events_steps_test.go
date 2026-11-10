//go:build acceptance

package acceptance

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	"github.com/cucumber/godog"
	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid/v5"

	"github.com/harvor-io/relay/internal/bus"
	"github.com/harvor-io/relay/internal/config"
	"github.com/harvor-io/relay/internal/database"
	"github.com/harvor-io/relay/internal/database/sqlite"
	"github.com/harvor-io/relay/internal/handlers"
	"github.com/harvor-io/relay/internal/middleware"
	"github.com/harvor-io/relay/internal/models"
	sqliterepo "github.com/harvor-io/relay/internal/repositories/sqlite"
	"github.com/harvor-io/relay/internal/services"
	"github.com/harvor-io/relay/pkg/encryptor/aes"
	secretssqlite "github.com/harvor-io/relay/pkg/secrets/sqlite"
)

// events is one scenario of features/events.feature: Relay's source and key
// services over real SQLite repositories and secret store, in a database of
// the scenario's own, behind the same request-signing check and routes the
// app serves. Only the eventbus, a boundary these rules don't reach, is
// mocked.
type events struct {
	dir     string
	db      *sql.DB
	sources services.SourceService
	keys    services.SourceKeyService
	router  http.Handler

	// sourceIDs are the scenario's sources by slug. issued are its keys by
	// name, with the secret each was issued with.
	sourceIDs map[string]uuid.UUID
	issued    map[string]issuedKey

	// response is what Relay answered when the scenario's event was published.
	response *httptest.ResponseRecorder
}

type issuedKey struct {
	sourceID uuid.UUID
	id       uuid.UUID
	secret   string
}

func initializeEventScenario(sc *godog.ScenarioContext) {
	w := &events{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		return ctx, w.setUp(ctx)
	})
	sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		return ctx, w.tearDown()
	})

	sc.Step(`^the "([^"]*)" source has an active key "([^"]*)"$`, w.sourceHasActiveKey)
	sc.Step(`^the "([^"]*)" source also has an active key "([^"]*)"$`, w.sourceHasActiveKey)
	sc.Step(`^the "([^"]*)" source is deactivated$`, w.sourceIsDeactivated)
	sc.Step(`^the "([^"]*)" key is deactivated$`, w.keyIsDeactivated)
	sc.Step(`^the (\S+) service publishes an? "([^"]*)" event signed with "([^"]*)"$`, w.servicePublishesEvent)
	sc.Step(`^the event is accepted under the topic "([^"]*)"$`, w.eventIsAcceptedUnderTopic)
	sc.Step(`^the event is accepted$`, w.eventIsAccepted)
	sc.Step(`^the event is refused$`, w.eventIsRefused)
}

// setUp gives the scenario a freshly migrated database and Relay's services
// over it, wired as `relay serve` wires them.
func (w *events) setUp(ctx context.Context) error {
	dir, err := os.MkdirTemp("", "relay-acceptance-")
	if err != nil {
		return err
	}
	w.dir = dir

	w.db, err = database.Open(ctx, &config.Config{
		DatabaseDriver: sqlite.DriverName,
		DatabaseURL:    "file:" + filepath.Join(dir, "relay.db"),
	})
	if err != nil {
		return err
	}
	migrator, err := database.NewMigrator(w.db, sqlite.DriverName)
	if err != nil {
		return err
	}
	if _, err := migrator.Up(ctx); err != nil {
		return err
	}

	secretStore, err := secretssqlite.NewStore(ctx, w.db)
	if err != nil {
		return err
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return err
	}
	enc, err := aes.New(key)
	if err != nil {
		return err
	}

	eventBus := &bus.BusMock{
		PublishFunc: func(context.Context, *models.Envelope) error { return nil },
	}
	sourceRepo := sqliterepo.NewSourceRepository(w.db)
	keyRepo := sqliterepo.NewSourceKeyRepository(w.db)
	w.sources = services.NewSourceService(sourceRepo, sqliterepo.NewEnvelopeRepository(w.db), keyRepo, secretStore, eventBus)
	w.keys = services.NewSourceKeyService(keyRepo, sourceRepo, secretStore, enc)

	router := chi.NewRouter()
	handlers.NewSourceHandler(w.sources, middleware.HMACAuth(w.sources, w.keys)).RegisterRoutes(router)
	w.router = router

	w.sourceIDs = map[string]uuid.UUID{}
	w.issued = map[string]issuedKey{}
	return nil
}

func (w *events) tearDown() error {
	if w.db != nil {
		_ = w.db.Close()
	}
	return os.RemoveAll(w.dir)
}

// source returns the scenario's source with slug, registering it first if
// the scenario hasn't yet.
func (w *events) source(ctx context.Context, slug string) (uuid.UUID, error) {
	if id, ok := w.sourceIDs[slug]; ok {
		return id, nil
	}
	source, err := w.sources.Create(ctx, services.CreateSourceInput{Name: slug, Slug: slug})
	if err != nil {
		return uuid.Nil, fmt.Errorf("register source %q: %w", slug, err)
	}
	w.sourceIDs[slug] = source.ID
	return source.ID, nil
}

func (w *events) sourceHasActiveKey(ctx context.Context, slug, name string) error {
	sourceID, err := w.source(ctx, slug)
	if err != nil {
		return err
	}
	key, secret, err := w.keys.Create(ctx, sourceID, services.CreateSourceKeyInput{Name: name})
	if err != nil {
		return fmt.Errorf("issue key %q: %w", name, err)
	}
	w.issued[name] = issuedKey{sourceID: sourceID, id: key.ID, secret: secret}
	return nil
}

func (w *events) sourceIsDeactivated(ctx context.Context, slug string) error {
	sourceID, err := w.source(ctx, slug)
	if err != nil {
		return err
	}
	_, err = w.sources.Deactivate(ctx, sourceID)
	return err
}

func (w *events) keyIsDeactivated(ctx context.Context, name string) error {
	key, ok := w.issued[name]
	if !ok {
		return fmt.Errorf("no key %q in this scenario", name)
	}
	_, err := w.keys.Deactivate(ctx, key.sourceID, key.id)
	return err
}

// servicePublishesEvent publishes an event of eventType to the source with
// slug, signed with the named key's secret, as a producer's service would.
func (w *events) servicePublishesEvent(slug, eventType, keyName string) error {
	key, ok := w.issued[keyName]
	if !ok {
		return fmt.Errorf("no key %q in this scenario", keyName)
	}
	body, err := json.Marshal(map[string]any{"type": eventType, "data": map[string]string{"example": "data"}})
	if err != nil {
		return err
	}
	mac := hmac.New(sha256.New, []byte(key.secret))
	mac.Write(body)

	request := httptest.NewRequest(http.MethodPost, "/sources/"+slug+"/events", strings.NewReader(string(body)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(middleware.SignatureHeader, hex.EncodeToString(mac.Sum(nil)))
	w.response = httptest.NewRecorder()
	w.router.ServeHTTP(w.response, request)
	return nil
}

func (w *events) eventIsAccepted() error {
	if w.response.Code != http.StatusCreated {
		return fmt.Errorf("the event was not accepted: %d %s", w.response.Code, w.response.Body.String())
	}
	return nil
}

func (w *events) eventIsAcceptedUnderTopic(topic string) error {
	if err := w.eventIsAccepted(); err != nil {
		return err
	}
	var envelope struct {
		Topic string `json:"topic"`
	}
	if err := json.Unmarshal(w.response.Body.Bytes(), &envelope); err != nil {
		return fmt.Errorf("decode accepted event: %w", err)
	}
	if envelope.Topic != topic {
		return fmt.Errorf("the event was accepted under the topic %q, want %q", envelope.Topic, topic)
	}
	return nil
}

// eventIsRefused checks Relay turned the event away as the caller's mistake,
// not that it failed: a server error would mean the rule went unchecked.
func (w *events) eventIsRefused() error {
	if w.response.Code < 400 || w.response.Code >= 500 {
		return fmt.Errorf("the event was not refused: %d %s", w.response.Code, w.response.Body.String())
	}
	return nil
}
