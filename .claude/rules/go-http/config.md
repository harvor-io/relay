---
paths:
  - "**/config/**/*.go"
  - "**/cmd/**/*.go"
  - "**/commands/**/*.go"
---

# Go HTTP projects: configuration

General conventions for Go HTTP API projects. Where this project's own code or
CLAUDE.md differs, follow the project.

## One config package and struct

- A `config` package holds a single `Config` struct, built by `config.New()`
  from environment variables. Nothing else reads the environment.
- Group related settings into sub-config structs (`HTTP`, `Database`, `Email`,
  `Payments`), with env var prefixes to match (`HTTP_*`, `DATABASE_*`,
  `EMAIL_*`, `PAYMENTS_*`).
- Each field's comment says what it controls and what happens when it's unset.
- Commands build the config once and pass each component the parts it needs.
  Components never read config globally.

## Defaults that make local setup simpler

- Provide sensible defaults as named constants, with a comment saying why.
  Typical ones are a localhost database URL for the local stack, `noop`
  providers instead of third-party ones, port `8080`, and `info` logging.
- The aim is for the project's documented local run command to work with no
  extra setup.
- Secrets and production-only values have no defaults.
- When a required setting is missing, the affected feature fails closed: its
  route responds `503`, or the command refuses to run. Never silently fall
  back to something insecure.
- Document every variable in the project's README.
