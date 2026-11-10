//go:build acceptance

package acceptance

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/cucumber/godog"
	"github.com/gofrs/uuid/v5"

	"github.com/harvor-io/relay/internal/middleware"
	"github.com/harvor-io/relay/internal/services"
)

// events is one scenario of features/events.feature.
type events struct {
	relay *relay

	// sourceIDs are the scenario's sources by slug. issued are its keys by
	// name, with the secret each was issued with.
	sourceIDs map[string]uuid.UUID
	issued    map[string]issuedKey

	// acceptedID is the ID of the event the scenario published beforehand.
	acceptedID uuid.UUID

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
		var err error
		w.relay, err = newRelay(ctx)
		w.sourceIDs = map[string]uuid.UUID{}
		w.issued = map[string]issuedKey{}
		w.acceptedID = uuid.Nil
		w.response = nil
		return ctx, err
	})
	sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		return ctx, w.relay.close()
	})

	sc.Step(`^the "([^"]*)" source has an active key "([^"]*)"$`, w.sourceHasActiveKey)
	sc.Step(`^the "([^"]*)" source also has an active key "([^"]*)"$`, w.sourceHasActiveKey)
	sc.Step(`^the "([^"]*)" source is deactivated$`, w.sourceIsDeactivated)
	sc.Step(`^the "([^"]*)" key is deactivated$`, w.keyIsDeactivated)
	sc.Step(`^the (\S+) service has published an? "([^"]*)" event$`, w.serviceHasPublishedEvent)
	sc.Step(`^the (\S+) service publishes an? "([^"]*)" event signed with "([^"]*)"$`, w.servicePublishesEvent)
	sc.Step(`^the (\S+) service publishes an? "([^"]*)" event with the same ID$`, w.servicePublishesEventWithSameID)
	sc.Step(`^the event is accepted under the topic "([^"]*)"$`, w.eventIsAcceptedUnderTopic)
	sc.Step(`^the event is accepted$`, w.eventIsAccepted)
	sc.Step(`^the event is refused$`, w.eventIsRefused)
	sc.Step(`^the accepted event is still an? "([^"]*)" event$`, w.acceptedEventIsStill)
}

// source returns the scenario's source with slug, registering it first if
// the scenario hasn't yet.
func (w *events) source(ctx context.Context, slug string) (uuid.UUID, error) {
	if id, ok := w.sourceIDs[slug]; ok {
		return id, nil
	}
	source, err := w.relay.sources.Create(ctx, services.CreateSourceInput{Name: slug, Slug: slug})
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
	key, secret, err := w.relay.keys.Create(ctx, sourceID, services.CreateSourceKeyInput{Name: name})
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
	_, err = w.relay.sources.Deactivate(ctx, sourceID)
	return err
}

func (w *events) keyIsDeactivated(ctx context.Context, name string) error {
	key, ok := w.issued[name]
	if !ok {
		return fmt.Errorf("no key %q in this scenario", name)
	}
	_, err := w.relay.keys.Deactivate(ctx, key.sourceID, key.id)
	return err
}

// servicePublishesEvent publishes an event of eventType to the source with
// slug, signed with the named key's secret, as a producer's service would.
func (w *events) servicePublishesEvent(slug, eventType, keyName string) error {
	return w.publish(slug, eventType, keyName, uuid.Nil)
}

// serviceHasPublishedEvent publishes an event of eventType to the source with
// slug, signed with one of its keys, and checks Relay accepted it.
func (w *events) serviceHasPublishedEvent(slug, eventType string) error {
	keyName, err := w.keyOf(slug)
	if err != nil {
		return err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	if err := w.publish(slug, eventType, keyName, id); err != nil {
		return err
	}
	if err := w.eventIsAccepted(); err != nil {
		return err
	}
	w.acceptedID = id
	return nil
}

// servicePublishesEventWithSameID publishes an event of eventType to the
// source with slug, signed with one of its keys, reusing the ID of the event
// the scenario published beforehand.
func (w *events) servicePublishesEventWithSameID(slug, eventType string) error {
	if w.acceptedID.IsNil() {
		return fmt.Errorf("no event was published beforehand")
	}
	keyName, err := w.keyOf(slug)
	if err != nil {
		return err
	}
	return w.publish(slug, eventType, keyName, w.acceptedID)
}

// keyOf returns the name of a key issued to the source with slug.
func (w *events) keyOf(slug string) (string, error) {
	sourceID, ok := w.sourceIDs[slug]
	if !ok {
		return "", fmt.Errorf("no source %q in this scenario", slug)
	}
	for name, key := range w.issued {
		if key.sourceID == sourceID {
			return name, nil
		}
	}
	return "", fmt.Errorf("no key for source %q in this scenario", slug)
}

// publish posts an event of eventType to the source with slug, signed with
// the named key's secret. A non-nil id is sent as the event's ID.
func (w *events) publish(slug, eventType, keyName string, id uuid.UUID) error {
	key, ok := w.issued[keyName]
	if !ok {
		return fmt.Errorf("no key %q in this scenario", keyName)
	}
	event := map[string]any{"type": eventType, "data": map[string]string{"example": "data"}}
	if !id.IsNil() {
		event["envelope_id"] = id.String()
	}
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	mac := hmac.New(sha256.New, []byte(key.secret))
	mac.Write(body)

	request := httptest.NewRequest(http.MethodPost, "/sources/"+slug+"/events", strings.NewReader(string(body)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(middleware.SignatureHeader, hex.EncodeToString(mac.Sum(nil)))
	w.response = httptest.NewRecorder()
	w.relay.router.ServeHTTP(w.response, request)
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

// acceptedEventIsStill checks the event the scenario published beforehand is
// stored exactly as it was accepted.
func (w *events) acceptedEventIsStill(ctx context.Context, eventType string) error {
	envelope, err := w.relay.envelopes.Get(ctx, w.acceptedID)
	if err != nil {
		return fmt.Errorf("get the accepted event: %w", err)
	}
	if envelope.Type != eventType {
		return fmt.Errorf("the accepted event is now a %q event, want %q", envelope.Type, eventType)
	}
	return nil
}
