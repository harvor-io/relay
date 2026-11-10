# Relay — working agreement

[CONTRIBUTING.md](CONTRIBUTING.md) covers the project layout, configuration,
running Relay, the checks CI enforces, and the steps for adding an endpoint.
The go-http rules in `.claude/rules/` and the global rules cover the rest; this
file holds only what's specific to Relay or where Relay departs from them.

## Working style

- **Do not spin up environments** — no running the server, `make run`, Docker,
  `curl` smoke tests, seeding databases, or running the end-to-end tests —
  unless the user explicitly asks. Write the tests the stability level calls
  for; just don't run the ones that need the stack.
- `go build ./...` and `make check` (`go fix`, `go fmt`, `go vet`,
  golangci-lint, and the Redocly lint) are fine and expected.

## Stability levels

Besides each capability doc, every operation in the OpenAPI spec under
`static/docs/` states its level in `x-stability`. Keep the two in step.

## Where Relay departs from the go-http rules

- **Migrations run on startup:** `relay serve` applies pending migrations
  before it serves, as well as through `relay migrate`. Platform engineers
  rely on it, so keep it.
