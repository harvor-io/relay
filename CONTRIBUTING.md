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

For a conceptual overview of how Relay works — sources, destinations,
subscriptions, and the event flow between them — see
[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Prerequisites

- [Go](https://go.dev/dl/) (the version in [go.mod](go.mod))
- [Docker](https://docs.docker.com/get-docker/) and Docker Compose (optional,
  for the containerized workflow)
- [golangci-lint](https://golangci-lint.run/welcome/install/) v2 (optional,
  for linting locally)

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
| [e2e/](e2e/) | Black-box Playwright tests that exercise the built binary over HTTP |

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

OpenAPI documentation, rendered with
[Stoplight Elements](https://stoplight.io/open-source/elements), is served at
<http://localhost:8080/docs/api>. The spec source lives in
[static/docs/](static/docs/) and is embedded into the binary, so there is no
bundling step — edit the YAML and restart.

## Tests and checks

Run the tests:

```sh
make test
```

Run everything CI enforces (`go fix`, `go fmt`, `go vet`, `golangci-lint`):

```sh
make check
```

CI additionally runs `go build ./...` and a Docker image build. Please make
sure `make test` and `make check` pass before opening a pull request.

### Black-box (Playwright) tests

[e2e/](e2e/) holds high-level tests that speak plain HTTP to a running Relay
server, exercising the happy path of each endpoint end to end. They don't
start a server themselves — start one first (`make up` or `make run`), then
point the suite at it. They need Node.js:

```sh
make up           # or: make run
make e2e-install  # once, or whenever e2e/package.json changes
make e2e          # defaults to http://localhost:8080/api/v1
```

Point the suite at a different instance with `BASE_URL`:

```sh
BASE_URL=http://localhost:9090/api/v1/ npm --prefix e2e test
```

Add a spec alongside the existing ones in [e2e/tests/](e2e/tests/) when a new
endpoint's happy path should be covered here too.

## Adding an endpoint

1. Add a handler in [internal/handlers/](internal/handlers/), following the
   existing pattern (a `*Handler` type with a `RegisterRoutes(chi.Router)`
   method).
2. Register it in [internal/commands/serve.go](internal/commands/serve.go).
3. Document it under [static/docs/](static/docs/): add a path file and wire it
   into [static/docs/api.yaml](static/docs/api.yaml).
4. Add a test alongside the handler.
