# Subscriptions

**Stability:** Prototype

A team that consumes events subscribes its destination to the topics it wants,
naming them exactly or by a pattern, so it chooses what it receives without
asking the producing team to change anything.

## What people can do

### Consumer team

- Subscribe a destination to a source's events, naming one or more topics it
  wants, such as `orders.order.created` and `orders.order.cancelled`.
- Subscribe with a wildcard pattern, such as every `order` event from the
  `orders` source, and receive new event types that match it without changing
  the subscription.
- Subscribe one destination to several sources, with a subscription for each.
- See which topics a destination is subscribed to.
- Subscribe to more topics or patterns from a source the destination is already
  subscribed to, keeping the ones it has.
- Unsubscribe from some of a subscription's topics or patterns, keeping the
  rest.
- Unsubscribe a destination from a source's events entirely.

## Learning

- Can a consumer team get exactly the events it wants by itself, without talking
  to the producing team?
- Which wildcard patterns do consumers actually reach for: a whole source, a
  resource's events, or something finer?

## Known gaps

- Subscribing doesn't deliver anything yet. Relay sets aside the events each
  active destination subscribes to, but delivering them to the destination is
  a separate capability that isn't built.
- There's no way to see which destinations subscribe to a source.
- A pattern like `orders.order.*` also matches an event whose type is exactly
  `order`, not only the event types that start with `order.`.
- If Relay fails partway through subscribing, changing topics, or
  unsubscribing, the request returns an error but the change may already be
  saved, and events may not be routed to match it. Deactivating and then
  reactivating the destination brings its routing back in line with its
  subscriptions.
- No sign-in or permissions: anyone who can reach Relay can subscribe any
  destination to any source's events, and remove any subscription. Run Relay
  where only trusted services and people can reach it.

## Rules

- [Subscriptions](../domain/rules/subscriptions.md)
- [Routing](../domain/rules/routing.md)

## Concepts

- [Subscription](../domain/concepts/subscription.md)
- [Topic](../domain/concepts/topic.md)
- [Destination](../domain/concepts/destination.md)
- [Source](../domain/concepts/source.md)
