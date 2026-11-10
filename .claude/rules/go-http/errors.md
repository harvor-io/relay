---
paths:
  - "**/service/**/*.go"
  - "**/services/**/*.go"
  - "**/handlers/**/*.go"
  - "**/http/**/*.go"
  - "**/api/**/*.go"
  - "**/server/**/*.go"
---

# Go HTTP projects: service errors

General conventions for Go HTTP API projects. Where this project's own code or
CLAUDE.md differs, follow the project.

## Services return explicit errors

- Export sentinel errors from the services package, named for the outcome:
  `var ErrAppointmentNotFound = errors.New("services: appointment not found")`.
  Shared ones cover cross-cutting outcomes, such as `ErrUnauthenticated` and
  `ErrForbidden`.
- Services translate repository errors into their own, for example
  `repositories.ErrNotFound` becomes `ErrAppointmentNotFound`. Handlers never
  key off repository or vendor errors.
- Something outside the caller's scope returns the same not-found error as
  something missing.
- To add detail without breaking `errors.Is`, wrap both:
  `fmt.Errorf("%w: %w", ErrInvalidInput, err)`. Wrap unexpected failures with
  context. They become 500s.
- Each method's doc comment names the errors it returns.

## Handlers key off them

- Each resource has a `writeXError(w, r, msg, err)` that uses an `errors.Is`
  switch to map each sentinel to a status and a client-safe message.
- The default case is a 500 with a generic message, and the real cause is
  logged.
- Never match on error strings.
