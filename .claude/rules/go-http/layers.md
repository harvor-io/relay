---
paths:
  - "**/*.go"
---

# Go HTTP projects: layers and boundaries

General conventions for Go HTTP API projects. Where this project's own code or
CLAUDE.md differs, follow the project.

## Layers

Handlers → services → boundary interfaces (repositories, external providers).

- **Services own the business logic and rules:** invariants, authorization and
  scoping, orchestration across repositories and providers, and transactions.
  Name methods in business terms where that fits (`Reschedule`, `Cancel`,
  `Approve`), not only CRUD verbs. Use one `XService` per area, built with
  `NewXService(...)` from interfaces, with `ctx` as the first parameter of
  every method.
- **Handlers are thin:** decode and validate the request, call a service, map
  its error, and render a resource. No business rules.
- **Repositories only persist.** See the repositories rule.

## Domain models and translation boundaries

Core logic operates on the project's own domain models. Anything crossing to
or from an external system is translated at that boundary, whether it's the
database, a vendor API, a webhook payload, or outbound HTTP. The boundary's
interface takes and returns domain models, and its implementation does the
translation. Vendor types never leak past their implementation package.

## Interfaces and implementations

- A boundary package named for the concept defines the interface in domain
  terms: `repositories`, `email`, `payments`, `storage`.
- Implementations live in subdirectories named for the technology or vendor:
  `repositories/postgres`, `email/smtp`, `email/noop`, `payments/stripe`.
  Only that subdirectory imports the vendor's SDK or driver.
- Each implementation asserts that it satisfies the interface:
  `var _ email.Sender = (*Sender)(nil)`.
- Services depend on the interface. The concrete implementation is chosen
  once, where the app is wired, usually from config.

Add an interface for a real boundary. A `noop` implementation is welcome when
it lets the app run locally without a vendor's credentials. Don't add
interfaces speculatively.
