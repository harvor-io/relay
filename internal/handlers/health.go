// Package handlers contains the HTTP handlers exposed by the API.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/alexliesenfeld/health"
	"github.com/go-chi/chi/v5"
)

// healthCheckTimeout bounds each check, inside the 1 second a Kubernetes
// probe waits by default.
const healthCheckTimeout = time.Second

// errUnhealthy is what a failed check reports. The endpoints are public, so
// the real cause is logged rather than returned.
var errUnhealthy = errors.New("unhealthy")

// HealthHandler serves the process's health checks, and only for what the
// process runs: one check for each of its servers and one for each
// dependency they use. Every server mounts the same HealthHandler, so any of
// them reports on all.
//
//   - /livez reports the servers alone, so a dependency outage never gets
//     the process restarted.
//   - /readyz reports the servers and their dependencies, so the process is
//     taken out of rotation while any is down.
//   - /healthz reports the same as /readyz, plus the external providers the
//     servers use. Relay uses none yet, so add an AddProvider (run in the
//     background, reported by /healthz alone) once it does.
//
// A dependency check also reports who provides it and how long its last
// check took.
//
// Checks are added while the servers are built, and Start must be called once
// they all have, before any server serves.
type HealthHandler struct {
	logger       *slog.Logger
	servers      []health.Check
	dependencies []health.Check
	live         http.Handler
	ready        http.Handler
	full         http.Handler

	mu        sync.Mutex
	provider  map[string]string
	latencies map[string]time.Duration
}

// NewHealthHandler constructs a HealthHandler with no checks. logger records
// why a check failed.
func NewHealthHandler(logger *slog.Logger) *HealthHandler {
	return &HealthHandler{
		logger:    logger,
		provider:  map[string]string{},
		latencies: map[string]time.Duration{},
	}
}

// AddServer adds a check for one of the process's servers, reported under
// name by every endpoint: up while check returns nil, and down when it
// returns an error.
func (h *HealthHandler) AddServer(name string, check func(ctx context.Context) error) {
	h.servers = append(h.servers, h.check(name, check))
}

// AddDependency adds a check for a dependency the servers use, provided by
// provider (such as "sqlite"), reported under name by /readyz and /healthz
// but not /livez: up while check returns nil, and down when it returns an
// error.
func (h *HealthHandler) AddDependency(name, provider string, check func(ctx context.Context) error) {
	h.provider[name] = provider
	h.dependencies = append(h.dependencies, h.check(name, h.timed(name, check)))
}

// check wraps check so a failure is logged, and reported only as unhealthy.
func (h *HealthHandler) check(name string, check func(ctx context.Context) error) health.Check {
	return health.Check{
		Name: name,
		Check: func(ctx context.Context) error {
			if err := check(ctx); err != nil {
				h.logger.Warn("health check failed", "check", name, "error", err)
				return errUnhealthy
			}
			return nil
		},
	}
}

// timed wraps check so how long each call takes, failed or not, is kept as
// name's latency.
func (h *HealthHandler) timed(name string, check func(ctx context.Context) error) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		start := time.Now()
		err := check(ctx)
		h.mu.Lock()
		h.latencies[name] = time.Since(start)
		h.mu.Unlock()
		return err
	}
}

// Start builds the checkers from the checks added so far.
func (h *HealthHandler) Start() {
	writer := health.WithResultWriter(healthResultWriter{h})
	h.live = health.NewHandler(health.NewChecker(
		health.WithTimeout(healthCheckTimeout),
		health.WithChecks(h.servers...),
	), writer)
	h.ready = health.NewHandler(health.NewChecker(
		health.WithTimeout(healthCheckTimeout),
		health.WithChecks(h.servers...),
		health.WithChecks(h.dependencies...),
	), writer)
	// With no external providers, /healthz reports exactly what /readyz does.
	h.full = h.ready
}

// RegisterRoutes mounts the handler's routes on r.
func (h *HealthHandler) RegisterRoutes(r chi.Router) {
	r.Get("/livez", h.GetLive)
	r.Get("/readyz", h.GetReady)
	r.Get("/healthz", h.GetHealth)
}

// GetLive runs the server checks and reports each one's status and the
// overall status: 200 when all are up, 503 when any is down. Results are
// cached for a second.
func (h *HealthHandler) GetLive(w http.ResponseWriter, r *http.Request) {
	h.live.ServeHTTP(w, r)
}

// GetReady runs the server and dependency checks and reports each one's
// status and the overall status: 200 when all are up, 503 when any is down.
// Results are cached for a second, so probes cannot hammer the dependencies.
func (h *HealthHandler) GetReady(w http.ResponseWriter, r *http.Request) {
	h.ready.ServeHTTP(w, r)
}

// GetHealth runs the server and dependency checks and reports each one's
// status and the overall status: 200 when all are up, 503 when any is down.
// Results are cached for a second, and shared with GetReady.
func (h *HealthHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
	h.full.ServeHTTP(w, r)
}

// healthResultWriter writes a result as the health library's JSON does, with
// each check's provider and last latency added where it has them.
type healthResultWriter struct {
	h *HealthHandler
}

// healthResult is the JSON body of every health endpoint.
type healthResult struct {
	Status  health.AvailabilityStatus    `json:"status"`
	Details map[string]healthCheckResult `json:"details,omitempty"`
}

// healthCheckResult is one check in a healthResult.
type healthCheckResult struct {
	Status    health.AvailabilityStatus `json:"status"`
	Provider  string                    `json:"provider,omitempty"`
	Timestamp time.Time                 `json:"timestamp,omitzero"`
	Error     string                    `json:"error,omitempty"`
	LatencyMS *int64                    `json:"latency_ms,omitempty"`
}

func (rw healthResultWriter) Write(result *health.CheckerResult, statusCode int, w http.ResponseWriter, _ *http.Request) error {
	body := healthResult{Status: result.Status, Details: make(map[string]healthCheckResult, len(result.Details))}
	rw.h.mu.Lock()
	for name, check := range result.Details {
		c := healthCheckResult{
			Status:    check.Status,
			Provider:  rw.h.provider[name],
			Timestamp: check.Timestamp,
		}
		if check.Error != nil {
			c.Error = check.Error.Error()
		}
		if latency, ok := rw.h.latencies[name]; ok {
			ms := latency.Milliseconds()
			c.LatencyMS = &ms
		}
		body.Details[name] = c
	}
	rw.h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(body)
}
