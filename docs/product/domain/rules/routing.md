# Routing rules

How events get from the sources that publish them to the destinations that
receive them.

## Producers and consumers don't know about each other

A source publishes without knowing which destinations receive its events, and a
destination subscribes without the source changing anything. Each team manages
its own source or destination.

- **Kind:** Core
- **Why:** this is what keeps producers independent of the systems and teams
  that consume their events.
- **Acceptance test:** None yet

## The message broker stays hidden

Relay routes events through a message broker of its own, but nobody who owns a
source or destination ever sees which broker it is, sees its queues or
exchanges, or configures it. A destination's kind, such as a RabbitMQ
destination, is a separate choice from the broker Relay uses inside.

- **Kind:** Core
- **Why:** so Relay can change its broker without anyone who uses it noticing.
- **Acceptance test:** None yet
