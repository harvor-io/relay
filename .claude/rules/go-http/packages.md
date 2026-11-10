---
paths:
  - "**/*.go"
  - "**/go.mod"
---

# Go HTTP projects: preferred packages

General conventions for Go HTTP API projects. Where this project's own code or
CLAUDE.md differs, follow the project.

- `github.com/go-chi/chi/v5`: routing and middleware
- `github.com/go-chi/render`: JSON responses and status codes
- `github.com/go-playground/validator/v10`: request validation through struct
  tags, with one process-wide validator
- `github.com/gofrs/uuid/v5`: IDs (UUIDv7)
- `github.com/exalynt/turn`: pagination
- `github.com/urfave/cli/v3`: the binary's commands
- `github.com/pressly/goose/v3`: SQL migrations, embedded in the binary
- `github.com/matryer/moq`: generated mocks for tests

No ORMs. For anything else, prefer the standard library, and ask before adding
a dependency.
