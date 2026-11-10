---
paths:
  - "**/repositories/**"
  - "**/repository/**"
  - "**/store/**"
  - "**/migrations/**"
  - "**/*.sql"
---

# Go HTTP projects: repositories and migrations

General conventions for Go HTTP API projects. Where this project's own code or
CLAUDE.md differs, follow the project.

## Repositories are dumb persistence layers

- No business logic, no authorization, no decisions. They store and retrieve
  domain models.
- Use common names: `Get`, `List`, `Create`, `Update`, `Upsert`, `Delete`, with
  qualifiers where needed (`GetByEmail`). Business verbs belong in services.
- `List` takes a query struct of filters plus a page selection. A filter left
  at its zero value is not applied.
- Work that spans several writes runs inside the service's transaction (a unit
  of work carried on `ctx`). The repository doesn't decide on transactions.

## Errors

- Repository errors (`ErrNotFound`, `ErrConflict`, …) are declared in the
  interface file, alongside the interface they belong to.
- Each implementation translates raw driver or vendor errors into them, such
  as `sql.ErrNoRows` and unique or foreign-key violations, so services can key
  off them.

## SQL and migrations

- Hand-written, parameterized SQL. No ORMs.
- Schema changes are goose SQL migrations (`NNNNN_description.sql`), embedded
  in the binary.
- Migrations run from an explicit command or deploy step, never on startup.

## Changing the schema

How to change it depends on the stability of the features that use the tables
involved (see the `stability` skill). The strictest of them applies.

- **Prototype and Alpha**: take the quickest path. Editing an existing
  migration in place and resetting the database is fine.
- **Beta and GA**: never edit a migration that may already have run. Add a new
  one, and keep it non-breaking for the code already deployed:
  - add columns as nullable or with a default
  - don't drop or rename a column or table the deployed code still uses.
    Expand, then contract: add the new shape, move the code over and backfill,
    and remove the old shape in a later migration
  - don't tighten a constraint until existing rows meet it
- A breaking schema change at Beta needs the user's direction. At GA, clarify
  the direction with the user first.
