# Subscriptions rules

What holds for every subscription a destination has.

## A subscription always names at least one topic

A subscription with no topics or patterns doesn't exist. A destination that
wants nothing from a source has no subscription to it, so removing a
subscription's last topic is refused: unsubscribing from the source entirely
is how that's done.

- **Kind:** Core
- **Why:** a subscription that names nothing subscribes to nothing, so there's
  no point keeping one, and nothing disappears as a side effect of removing a
  topic.
- **Acceptance test:** None yet

## One subscription per destination and source

A destination has at most one subscription to each source. To want more of a
source's topics, the destination adds them to the subscription it has.

- **Kind:** Core
- **Acceptance test:** None yet

## Topics belong to the subscription's source

Every topic a subscription names starts with its source's slug, such as
`orders.order.created` for the `orders` source. A topic the source hasn't
published yet can still be named.

- **Kind:** Core
- **Why:** a subscription covers one source, and a consumer can subscribe
  before the producer starts publishing an event type.
- **Acceptance test:** None yet

## A wildcard ends a pattern

A pattern ends in `*`, which matches one or more further parts of a topic, and
`*` can't appear anywhere else. `orders.order.*` matches every `order` event
from `orders`, such as `orders.order.created`, and `orders.*` matches every
event from `orders`.

- **Kind:** Core
- **Acceptance test:** None yet

## Subscriptions outlive their source

Deleting a source leaves the subscriptions to it in place, still naming its
topics. A new source given the same slug publishes those same topics, so its
events match them.

- **Kind:** Edge case
- **Acceptance test:** None yet
