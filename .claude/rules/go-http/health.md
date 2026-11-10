---
paths:
  - "**/*health*.go"
  - "**/handlers/**/*.go"
  - "**/http/**/*.go"
  - "**/server/**/*.go"
  - "**/middleware/**/*.go"
  - "**/cmd/**/*.go"
  - "**/commands/**/*.go"
  - "**/provider*.go"
  - "**/.kube/**"
  - "**/compose*.y*ml"
  - "**/Dockerfile*"
---

# Go HTTP projects: health checks

General conventions for Go HTTP API projects. Where this project's own code or
CLAUDE.md differs, follow the project.

## When

Add the health endpoints once any feature in the project reaches Alpha (see
the `stability` skill), or sooner if the app is deployed somewhere that probes
it.

## Three endpoints

Each answers a different question, so an outage only does the damage it
should:

| Endpoint | Reports | Used by | So that |
| --- | --- | --- | --- |
| `GET /livez` | the servers the process runs | the liveness probe | a dependency outage never gets the process restarted |
| `GET /readyz` | the servers and their dependencies | the readiness probe, the load balancer's health check, and Compose's `healthcheck` | the process leaves rotation while any is down |
| `GET /healthz` | the servers, their dependencies, and the external providers they use | people and monitoring | a provider outage shows, but never takes the process out of rotation |

- Mount them under the API's version prefix with the other routes
  (`/v1/livez`).
- They're public, with no authentication, so probes reach them without a
  token. Say so in a comment where they're mounted.
- `200` when every check is up, `503` when any is down or hasn't run yet.
- Never point the liveness probe at `/readyz` or `/healthz`.

## What gets a check

Only what the process actually runs. A server, dependency, or provider that
isn't running or configured has no check.

- **Servers**: one per server the process runs, named for it (`http`, `wss`).
  It's down once that server stops serving, such as while it drains on
  shutdown. Every server mounts the same health handler, so any port reports
  on all of them.
- **Dependencies**: what the servers can't serve without, such as the
  database. One check each, named for its role (`database`), not the
  technology. When a dependency has several connections, the check pings them
  all and joins the errors.
- **External providers**: third-party services behind a boundary interface,
  named for the boundary (`payments`, `email`, `identity`, `storage`). Each
  implementation that reaches an external service has a
  `Ping(ctx context.Context) error` that makes the cheapest read its
  credentials already allow, so it proves the service answers and accepts the
  key. Its doc comment says what it calls.
- `noop`, memory, and local implementations reach nothing, so they have no
  `Ping` and no check. Don't add `Ping` to the boundary interface. Where the
  app is wired, a small `pinger` interface picks up the implementations that
  have one.

## How checks run

Use `github.com/alexliesenfeld/health`, behind one `HealthHandler` in the HTTP
handlers package:

- `AddServer(name, check)`, `AddDependency(name, provider, check)`, and
  `AddProvider(name, provider, check)` add checks while the servers are built.
  `provider` is who provides it (`postgres`, `stripe`).
- `Start()` builds the checkers once every check is added, before any server
  serves.

Timing:

- Server and dependency checks run when the endpoint is called, each bounded
  at 1 second, inside the 1 second a Kubernetes probe waits by default.
  Results are cached for a second so probes can't hammer the dependencies.
- Provider checks run in the background every 30 seconds, with a 5 second
  timeout. `/healthz` reports their last result, so a request never reaches a
  provider itself. Until a provider's first check finishes, it's `unknown`
  and `/healthz` responds `503`.

## Response body

Every endpoint returns the same shape, a top-level `status` and one entry per
check under `details`:

```json
{
  "status": "up",
  "details": {
    "http": { "status": "up", "timestamp": "2026-10-15T20:00:00Z" },
    "database": {
      "status": "up",
      "provider": "postgres",
      "timestamp": "2026-10-15T20:00:00Z",
      "latency_ms": 2
    },
    "payments": {
      "status": "down",
      "provider": "stripe",
      "timestamp": "2026-10-15T19:59:41Z",
      "latency_ms": 5000,
      "error": "unhealthy"
    }
  }
}
```

- `status` is `up`, `down`, or `unknown`.
- Dependencies and providers carry `provider` and `latency_ms`, how long their
  last check took, passed or failed.
- The endpoints are public, so a failed check reports only `"unhealthy"`. The
  real cause is logged at Warn with the check's name, and never returned:
  no error text, hosts, bucket names, or keys.
- This body replaces the REST API rule's `{"error": ...}` body on these
  endpoints, including their `503`s.
- Document the three endpoints and the body in the OpenAPI spec and the
  README like any other endpoint.

## Probes and telemetry

- Kubernetes: `livenessProbe` on `/livez`, `readinessProbe` on `/readyz`. The
  load balancer's health check and Compose's `healthcheck` use `/readyz`.
- Probe requests get no spans. Filter them out of the HTTP instrumentation
  (`otelhttp.WithFilter`), since orchestrators call them every few seconds
  and they would crowd out every other trace.
