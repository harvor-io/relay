# Sources

**Stability:** Alpha

A team that produces events registers a source in Relay and issues keys for it,
so its services can publish events without knowing where they end up.

## What people can do

### Producer team

- Register a source with a name, an optional description, and an optional slug.
  Relay makes up a slug from the name when none is given.
- Rename a source or change its description. Its slug stays the same.
- Deactivate a source to stop it accepting events, keeping its keys and
  history, and reactivate it later.
- Delete a source it no longer needs.
- Issue a named key for a source, and copy its secret, which is shown only once.
- Rotate keys without downtime: issue a new key, move services onto it, then
  deactivate or delete the old one.
- Deactivate a key to stop it being accepted, reactivate it, or delete it.

## Known gaps

- No sign-in or permissions: anyone who can reach Relay can see, change, and
  delete every source, and issue keys for any of them. Run Relay where only
  trusted services and people can reach it.
- A source's slug can't be changed after it's registered: register a new source
  instead.
- Sources can only be managed one call at a time. There's no way yet to declare
  them in a configuration file kept with the application.

## Rules

- [Events](../domain/rules/events.md)

## Concepts

- [Source](../domain/concepts/source.md)
- [Source key](../domain/concepts/source-key.md)
