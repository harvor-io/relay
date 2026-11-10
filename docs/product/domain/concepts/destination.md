# Destination

A registered target Relay delivers events to, such as a webhook URL. It's owned
by the team that consumes the events.

## Details

A destination has a name, an optional description, and a kind that says how
Relay delivers to it, along with the settings that kind needs. For a webhook,
those are its URL and, optionally, how deliveries are signed. The kind and its
settings are fixed once the destination is registered. A destination receives
the [events](event.md) whose [topics](topic.md) it has a
[subscription](subscription.md) to.

## Lifecycle

Active ⇄ Inactive → Deleted

A destination starts active. While inactive, nothing is delivered to it.

## Not to be confused with

- [Source](source.md): where events come from, not where they go.
- The message broker Relay uses inside: a RabbitMQ destination is somewhere
  Relay delivers to, not how Relay routes events.
