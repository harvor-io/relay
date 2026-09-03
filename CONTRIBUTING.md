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
| [internal/commands/](internal/commands/) | `serve` command: router setup and HTTP server |
| [internal/config/](internal/config/) | Runtime configuration, loaded from the environment |
| [internal/handlers/](internal/handlers/) | HTTP handlers (`chi` router, `chi/render` responses) |
| [internal/middleware/](internal/middleware/) | HTTP middleware (e.g. UUIDv7 request IDs) |
| [static/](static/) | OpenAPI spec, embedded into the binary with `embed.FS` |

### Configuration

Configuration is read from environment variables. Currently the only setting
is `PORT` (default `8080`).

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

## Adding an endpoint

1. Add a handler in [internal/handlers/](internal/handlers/), following the
   existing pattern (a `*Handler` type with a `RegisterRoutes(chi.Router)`
   method).
2. Register it in [internal/commands/serve.go](internal/commands/serve.go).
3. Document it under [static/docs/](static/docs/): add a path file and wire it
   into [static/docs/api.yaml](static/docs/api.yaml).
4. Add a test alongside the handler.
