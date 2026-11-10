# Sources rules

What holds for every source registered with Relay.

## A slug is unique across Relay

No two sources share a slug. A slug in use can't be taken again, and a slug
Relay makes up from a name steps aside for one in use, with a number added.

- **Kind:** Core
- **Why:** the slug starts every topic a source publishes to, so two sources
  with one slug would publish into each other's topics.
- **Acceptance test:** [sources.feature › A slug is unique across Relay](../../../../tests/acceptance/features/sources.feature)

## A slug never changes

Once a source is registered, its slug stays the same, even when the source is
renamed.

- **Kind:** Core
- **Acceptance test:** [sources.feature › A slug never changes](../../../../tests/acceptance/features/sources.feature)
