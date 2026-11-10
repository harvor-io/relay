# Contributing to Relay

Thanks for your interest in contributing to Relay!

## Scope of this document

General Harvor contribution guidelines — how we use issues and discussions,
branch and commit conventions, the pull request process, code quality
expectations, security reporting, and licensing — live in the Harvor
community repository:

<https://github.com/harvor-io/community/blob/main/CONTRIBUTING.md>

Please read that first. **This document only covers the specifics of working
in the Relay repository**: how the service is structured, how to run it
locally, and how to run the checks CI enforces.

For what Relay does — sources, destinations, subscriptions, and the rules
between them — see the [product docs](docs/product/README.md). For how it's
built, including the event flow, tradeoffs, and gotchas to read before changing
something, see the [engineering docs](docs/engineering/README.md).

## Prerequisites

- [Go](https://go.dev/dl/) (the version in [go.mod](go.mod))
- [Docker](https://docs.docker.com/get-docker/) and Docker Compose (optional,
  for the containerized workflow)
- [golangci-lint](https://golangci-lint.run/welcome/install/) v2 (optional,
  for linting locally)
- [Node.js](https://nodejs.org/) (for the end-to-end tests and the docs lint;
  run `npm ci` once)

## Project layout

Relay is a single Go binary. The CLI entrypoint is [cmd/relay](cmd/relay),
built with [urfave/cli](https://github.com/urfave/cli); `relay serve` starts
the HTTP API.

| Path | Purpose |
|---|---|
| [cmd/relay/](cmd/relay/) | CLI entrypoint and subcommand wiring |
| [internal/commands/](internal/commands/) | `serve` and `migrate` commands |
| [internal/config/](internal/config/) | Runtime configuration, loaded from the environment |
| [internal/handlers/](internal/handlers/) | HTTP handlers (`chi` router, `chi/render` responses) |
| [internal/middleware/](internal/middleware/) | HTTP middleware (e.g. UUIDv7 request IDs) |
| [internal/models/](internal/models/) | Core domain entities (`Source`, `Destination`, `Envelope`) |
| [internal/database/](internal/database/) | Connection setup (`sqlite`, `libsql` sub-packages) and the migration runner (`goose`) |
| [internal/database/migrations/](internal/database/migrations/) | SQL migration files, embedded with `embed.FS`, one sub-directory per engine |
| [internal/repositories/](internal/repositories/) | Persistence interfaces and their SQLite implementations |
| [static/](static/) | OpenAPI spec, embedded into the binary with `embed.FS` |
| [tests/e2e/](tests/e2e/) | Black-box Playwright tests: one endpoint's happy path each, over HTTP |
| [tests/acceptance/](tests/acceptance/) | Business rules in Gherkin (`features/`), checked with godog |

### Configuration

Configuration is read from environment variables:

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `DATABASE_DRIVER` | `sqlite` | Database driver: `sqlite` (embedded, pure-Go) or `libsql` (hosted libSQL / Turso) |
| `DATABASE_URL` | `file:relay.db` | Data source: a local `file:` DSN or a hosted libSQL URL (e.g. Turso `libsql://…`) |
| `DATABASE_AUTH_TOKEN` | _(empty)_ | Auth token for a hosted libSQL database; unused for local files |
| `LOG_LEVEL` | `info` | Minimum structured-log severity: `debug`, `info`, `warn`, or `error` |
| `LOG_FORMAT` | `json` | Log handler: `json` (machine-readable) or `text` (human-readable) |
| `SECRETS_DATABASE_DRIVER` | `sqlite` | Database driver for the secret store; same options as `DATABASE_DRIVER` |
| `SECRETS_DATABASE_URL` | value of `DATABASE_URL` | Data source for the secret store; set to keep secrets in a dedicated database |
| `SECRETS_ENCRYPTION_KEY` | _(none)_ | Key used to encrypt secrets at rest; must be base64-encoded 32 random bytes (e.g. `openssl rand -base64 32`) |
| `EVENTBUS_DRIVER` | `rabbitmq` | Eventbus backend; `rabbitmq` is the only driver currently supported |
| `RABBITMQ_URI` | `amqp://guest:guest@localhost:5672/` | AMQP connection URI for the RabbitMQ broker that backs the eventbus |

### Database migrations

Schema migrations are [goose](https://github.com/pressly/goose) SQL files under
[internal/database/migrations/](internal/database/migrations/), grouped by engine
(`internal/database/migrations/sqlite/` is shared by the `sqlite` and `libsql`
drivers). They are embedded into the binary.

```sh
go run ./cmd/relay migrate status   # list migrations and their state
go run ./cmd/relay migrate up       # apply all pending migrations
go run ./cmd/relay migrate down     # roll back the most recent migration
```

`relay serve` also applies pending migrations on startup. To add one, create
the next `NNNNN_name.sql` file in the engine directory with `-- +goose Up` and
`-- +goose Down` sections.

## Running locally

### Without Docker

```sh
make run
```

The server listens on `http://localhost:8080`. Use a different port with:

```sh
PORT=9090 make run
```

### With Docker Compose

```sh
cp compose.override.example.yml compose.override.yml
make build
```

The API is published on port 8080. Follow logs with `make logs`; stop the
stack with `make down`.

### Check the service

```sh
curl -i http://localhost:8080/api/v1/healthz
```

### API documentation

OpenAPI documentation, rendered with [Scalar](https://scalar.com), is served
at <http://localhost:8080/docs/api>. The spec source lives in
[static/docs/](static/docs/) and is embedded into the binary. Relay bundles it
into one document when it starts, so edit the YAML and restart.

| Route | What it serves |
| --- | --- |
| `GET /docs/api` | The spec rendered with Scalar, with a stability mark on each endpoint below GA and a maturity ring on each tag. |
| `GET /docs/openapi.yaml` | The spec bundled into one document: what the docs page renders. Each operation's `x-stability` becomes an `x-badges` badge here. |
| `GET /docs/*` | The raw spec files, starting at `/docs/api.yaml`. |

Every operation states its
[stability level](https://readme.exalynt.com/how-it-works/stability-levels)
in an `x-stability` field: `prototype`, `alpha`, `beta`, or `ga`. Relay won't
start if an operation is missing one or names an unknown level. Keep each
operation's level in step with its capability's level in
[docs/product/capabilities/](docs/product/capabilities/).
[stability-marks.js](static/docs/stability-marks.js) and
[stability-marks.css](static/docs/stability-marks.css) draw the marks and
rings. They're copied from Exalynt's API docs, so change them there first.

Lint the spec with Redocly, which CI also runs:

```sh
make docs-lint
```

The rules are in [redocly.yaml](redocly.yaml), and the deliberate exceptions,
each with its reason, are in [.redocly.lint-ignore.yaml](.redocly.lint-ignore.yaml).

## Tests and checks

Relay is at Alpha, so it has the two kinds of test Alpha calls for:

- **End-to-end tests** in [tests/e2e/](tests/e2e/): Playwright, black box, one
  endpoint's happy path per test, called over HTTP.
- **Acceptance tests** in [tests/acceptance/](tests/acceptance/): the business
  rules in the [product docs](docs/product/README.md), written in Gherkin in
  `features/` and checked with [godog](https://github.com/cucumber/godog). The
  `.feature` files are the specification, so a scenario changes only when the
  rule it states does.

Unit and integration tests start at Beta.

Run the acceptance tests. They need nothing else running: each scenario gets
its own SQLite database, and the eventbus is mocked.

```sh
make acceptance-test   # also: make test
```

Run the end-to-end tests. They need RabbitMQ on `localhost:5672`, which the
example override publishes. Playwright builds and starts Relay itself, or uses
one already running on port 8080 (`make up`, `make run`):

```sh
docker compose -f compose.yml -f compose.override.example.yml up -d --wait rabbitmq
make e2e-test
```

Point the suite at a different instance with `BASE_URL`:

```sh
BASE_URL=http://localhost:9090/api/v1/ npx playwright test tests/e2e
```

Run everything else CI enforces (`go fix`, `go fmt`, `go vet`, `golangci-lint`,
and the Redocly docs lint):

```sh
make check
```

CI additionally runs `go build ./...` and a Docker image build. Please make
sure the tests and `make check` pass before opening a pull request.

Mocks are generated with [moq](https://github.com/matryer/moq) from
`//go:generate` directives and committed. Regenerate them with
`go generate ./...` after changing a mocked interface.

## Adding an endpoint

1. Add a handler in [internal/handlers/](internal/handlers/), following the
   existing pattern (a `*Handler` type with a `RegisterRoutes(chi.Router)`
   method).
2. Register it in [internal/commands/serve.go](internal/commands/serve.go).
3. Document it under [static/docs/](static/docs/): add a path file, give each
   operation an `x-stability` level, and wire it into
   [static/docs/api.yaml](static/docs/api.yaml).
4. Add an end-to-end test of its happy path in [tests/e2e/](tests/e2e/).
