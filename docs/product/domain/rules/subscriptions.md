# Subscriptions rules

What holds for every subscription a destination has.

## A subscription names one topic or pattern

A subscription links a destination to a single topic, such as
`orders.order.created`, or a single pattern. A destination that wants several
has a subscription for each, and subscribing to one it already has changes
nothing.

- **Kind:** Core
- **Why:** events are routed by topic alone, so a subscription needs nothing
  else to say what a destination wants.
- **Acceptance test:** None yet

## A wildcard ends a pattern

A pattern ends in `*`, which matches one or more further parts of a topic, and
`*` can't appear anywhere else. `orders.order.*` matches every `order` event
from `orders`, such as `orders.order.created`, `orders.*` matches every event
from `orders`, and `*` alone matches every event from every source.

- **Kind:** Core
- **Acceptance test:** None yet

## Subscribing doesn't need the source

A destination can subscribe to a topic whose source isn't registered yet, or
whose event type hasn't been published yet. Only the topic's shape is checked:
a source's slug, then an event type, optionally ending in `*`.

- **Kind:** Core
- **Why:** a consumer can subscribe before the producer starts publishing, and
  neither team has to know the other's setup.
- **Acceptance test:** None yet

## Subscriptions outlive their source

Deleting a source leaves the subscriptions to its topics in place. A new source
given the same slug publishes those same topics, so its events match them.
Deleting a destination removes all of its subscriptions.

- **Kind:** Edge case
- **Why:** a subscription is to a topic, not to a source, and a source's slug,
  and so its topics, never change.
- **Acceptance test:** None yet

## Replacing a destination's subscriptions is all or nothing

A destination's subscriptions can be replaced with a new set in one step. If
any topic in the set isn't valid, none of them change.

- **Kind:** Edge case
- **Acceptance test:** None yet
