# Sources

Sources, their keys, and the signed events endpoint they publish through.

**Code:** `internal/services/sources.go` and `source_keys.go`,
`internal/middleware/source_event_auth.go` (`HMACAuth`), and
`internal/handlers/sources.go` and `source_keys.go`.

## Gotchas

- The events route resolves its `{id}` as either a source ID or a slug. Every
  other source and key route accepts only the ID.
- `HMACAuth` checks the signature before `CreateEvent` checks whether the source
  is active, so an inactive source still answers 401 for a bad signature and
  422 only for a good one.
- A key that belongs to another source answers 404, the same as one that doesn't
  exist, so callers can't probe for key IDs.
- The schema has no foreign keys, so deleting a source leaves its keys, their
  secrets, and its envelopes behind. A source recreated with the same explicit
  ID picks the old keys back up, and they're still active.

## Known gaps

- **Deleting a source doesn't delete what belongs to it:** its keys, key secrets,
  and envelopes stay in the databases, unreachable through the API. **Until:**
  not decided.
