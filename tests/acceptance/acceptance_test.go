//go:build acceptance

// Package acceptance holds Relay's business rules, written in Gherkin in
// features/, and the step definitions that check them.
package acceptance

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestAcceptance runs every feature in features/. Run it with
// go test -tags acceptance -run '^TestAcceptance$' ./tests/acceptance.
func TestAcceptance(t *testing.T) {
	// Each feature runs as a suite of its own, with its own steps, so two
	// features can phrase a step alike without one taking the other's.
	features, err := filepath.Glob("features/*.feature")
	if err != nil {
		t.Fatalf("find features: %v", err)
	}
	for _, path := range features {
		name := strings.TrimSuffix(filepath.Base(path), ".feature")
		t.Run(name, func(t *testing.T) {
			initialize, ok := scenarioInitializers[name]
			if !ok {
				t.Fatalf("no steps registered for features/%s.feature", name)
			}
			suite := godog.TestSuite{
				ScenarioInitializer: initialize,
				Options: &godog.Options{
					Format:   "pretty",
					Paths:    []string{path},
					Strict:   true,
					TestingT: t,
				},
			}
			if suite.Run() != 0 {
				t.Fatal("acceptance tests failed")
			}
		})
	}
}

// scenarioInitializers registers the steps of each feature, by its file's
// name.
var scenarioInitializers = map[string]func(*godog.ScenarioContext){
	"events": initializeEventScenario,
}
