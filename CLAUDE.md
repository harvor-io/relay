# Relay — working agreement

## Working style

- **Do not write tests** unless the user explicitly asks for them in the request.
- **Do not spin up environments** — no running the server, `make run`, Docker,
  `curl` smoke tests, or seeding databases — unless the user explicitly asks.
- Building (`go build ./...`), `go vet`, and `gofmt` are fine and expected.

## Project layout

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full layout. In short:

| Path | Purpose |
|---|---|
| `cmd/relay/` | CLI entrypoint (`urfave/cli`); `relay serve` starts the API |
| `internal/commands/` | `serve` and `migrate` subcommands and their wiring |
| `internal/config/` | Runtime configuration from the environment |
| `internal/handlers/` | HTTP handlers (`chi` router, `chi/render`); translate service errors to HTTP responses and models to explicit resource structs |
| `internal/services/` | Use-case layer: validation, coordination, well-defined domain errors; never speaks HTTP |
| `internal/repositories/` | Persistence interfaces and their SQLite implementations |
| `internal/models/` | Core domain entities (`Source`, `Destination`, `Envelope`) |
| `internal/database/` | Connection setup and the `goose` migration runner |
| `static/docs/` | OpenAPI spec, embedded into the binary |

## Conventions

- Handlers return only the resource(s) as needed for a RESTful API — no
  metadata (counts, pagination envelopes) in response bodies.
- Services return specific, comparable errors (`errors.Is`); handlers map those
  onto status codes.
- Error response bodies are `{"error": "message"}` (the `Error` schema).
- Document new endpoints under `static/docs/` and wire them into
  `static/docs/api.yaml`.
