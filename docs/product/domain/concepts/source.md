# Source

A registered origin allowed to publish events into Relay, such as `orders`. It's
owned by the team that produces the events.

## Details

A source has a name, an optional description, and a slug: a short, URL-safe name
that's unique across Relay and never changes. The slug is the first part of
every [topic](topic.md) the source publishes to. A source has one or more
[source keys](source-key.md) its services sign [events](event.md) with.

## Lifecycle

Active ⇄ Inactive → Deleted

A source starts active. Its owner can deactivate it to pause its events and
reactivate it later, or delete it.

## Not to be confused with

- [Destination](destination.md): where events are delivered, not where they
  come from.
