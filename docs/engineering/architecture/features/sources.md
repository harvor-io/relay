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
- The schema has no foreign keys, so `sourceService.Delete` deletes the source's
  keys and their secrets itself, after the source row. Like deleting one key,
  that's best effort: a failure partway leaves keys or secrets nobody can reach.
- Signature verification goes through `SourceKeyRepository.ListBySource`, which
  is unpaginated on purpose. Don't swap it for the paginated `List`, or only
  the first page of keys would be checked.
- An event type's limit depends on the source's slug: the topic,
  `<slug>.<type>`, is capped at 255 bytes because RabbitMQ silently truncates
  longer routing keys.

## Known gaps

- **Deleting a source leaves its envelopes:** they stay in the database,
  unreachable through the API. **Until:** event retention is built, which
  clears them out by age.
- **Signed requests can be replayed:** the signature covers only the body, with
  no timestamp, so a captured request can be sent again. An event ID already
  used is refused, but one without an ID is accepted again. **Until:**
  Publishing events reaches Beta.
