//go:build acceptance

package acceptance

import (
	"context"
	"errors"
	"fmt"

	"github.com/cucumber/godog"
	"github.com/gofrs/uuid/v5"

	"github.com/harvor-io/relay/internal/models"
	"github.com/harvor-io/relay/internal/services"
)

// sources is one scenario of features/sources.feature.
type sources struct {
	relay *relay

	// registered is the scenario's sources by slug.
	registered map[string]uuid.UUID

	// source and err are what Relay answered when the scenario registered or
	// changed a source.
	source *models.Source
	err    error
}

func initializeSourceScenario(sc *godog.ScenarioContext) {
	w := &sources{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		var err error
		w.relay, err = newRelay(ctx)
		w.registered = map[string]uuid.UUID{}
		w.source, w.err = nil, nil
		return ctx, err
	})
	sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		return ctx, w.relay.close()
	})

	sc.Step(`^the "([^"]*)" source is registered$`, w.sourceIsRegistered)
	sc.Step(`^a source is registered with the slug "([^"]*)"$`, w.registerWithSlug)
	sc.Step(`^a source named "([^"]*)" is registered without a slug$`, w.registerWithoutSlug)
	sc.Step(`^the source is renamed "([^"]*)"$`, w.renameSource)
	sc.Step(`^the source is refused$`, w.sourceIsRefused)
	sc.Step(`^it is registered with the slug "([^"]*)"$`, w.registeredWithSlug)
	sc.Step(`^its slug is still "([^"]*)"$`, w.slugIsStill)
}

func (w *sources) sourceIsRegistered(ctx context.Context, slug string) error {
	source, err := w.relay.sources.Create(ctx, services.CreateSourceInput{Name: slug, Slug: slug})
	if err != nil {
		return fmt.Errorf("register source %q: %w", slug, err)
	}
	w.registered[slug] = source.ID
	return nil
}

func (w *sources) registerWithSlug(ctx context.Context, slug string) error {
	w.source, w.err = w.relay.sources.Create(ctx, services.CreateSourceInput{Name: "Another " + slug, Slug: slug})
	return nil
}

func (w *sources) registerWithoutSlug(ctx context.Context, name string) error {
	w.source, w.err = w.relay.sources.Create(ctx, services.CreateSourceInput{Name: name})
	return nil
}

// renameSource renames the scenario's only registered source.
func (w *sources) renameSource(ctx context.Context, name string) error {
	if len(w.registered) != 1 {
		return fmt.Errorf("want one registered source to rename, have %d", len(w.registered))
	}
	for _, id := range w.registered {
		w.source, w.err = w.relay.sources.Update(ctx, id, services.UpdateSourceInput{Name: &name})
	}
	return w.err
}

// sourceIsRefused checks Relay turned the source away because its slug is
// taken, not that it failed: any other error would mean the rule went
// unchecked.
func (w *sources) sourceIsRefused() error {
	if !errors.Is(w.err, services.ErrSourceSlugTaken) {
		return fmt.Errorf("the source was not refused for its slug: %v", w.err)
	}
	return nil
}

func (w *sources) registeredWithSlug(slug string) error {
	if w.err != nil {
		return fmt.Errorf("the source was not registered: %w", w.err)
	}
	if w.source.Slug != slug {
		return fmt.Errorf("the source was registered with the slug %q, want %q", w.source.Slug, slug)
	}
	return nil
}

func (w *sources) slugIsStill(ctx context.Context, slug string) error {
	source, err := w.relay.sources.Get(ctx, w.source.ID)
	if err != nil {
		return err
	}
	if source.Slug != slug {
		return fmt.Errorf("the source's slug is %q, want %q", source.Slug, slug)
	}
	return nil
}
