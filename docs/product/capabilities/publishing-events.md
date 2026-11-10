# Publishing events

**Stability:** Alpha

A producing service publishes events to its source over HTTP, so the event
reaches whoever needs it without the producer knowing who that is.

## What people can do

### Producer service

- Publish an event to its source, addressed by the source's slug or ID, signed
  with one of the source's keys.
- Say what kind of event it is with an event type, such as `order.created`.
- Give the event its own ID, so a retry can't be recorded twice.

## Known gaps

- Events aren't delivered anywhere yet. Relay records and accepts them, but no
  destination receives them until subscriptions and delivery are built.
- Events are kept indefinitely. Clearing out events older than a maximum age
  isn't built yet.
- Accepted events can't be looked up yet: there's no way to see what a source
  has published.
- HTTP is the only way to publish. gRPC is planned.
- If Relay fails partway through accepting an event, the producer gets an
  error, but the event may already be recorded. Retrying with the same event ID
  is then rejected as a duplicate, and the event may never be routed.
- If Relay loses its connection to its internal message broker, it rejects every
  event until it's restarted.

## Rules

- [Events](../domain/rules/events.md)
- [Routing](../domain/rules/routing.md)

## Concepts

- [Event](../domain/concepts/event.md)
- [Topic](../domain/concepts/topic.md)
- [Source](../domain/concepts/source.md)
- [Source key](../domain/concepts/source-key.md)
