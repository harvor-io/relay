# Destinations

**Stability:** Prototype

A team that consumes events registers a destination in Relay, saying where and
how events should be delivered to it, without the producers knowing it exists.

## What people can do

### Consumer team

- Register a webhook destination with a name, an optional description, and the
  URL events should be sent to.
- Have deliveries signed, by giving a signing secret and saying how the
  signature and timestamp are sent. The secret is stored securely and never
  shown back.
- Rename a destination or change its description.
- Deactivate a destination to stop deliveries to it, keeping it registered, and
  reactivate it later.
- Delete a destination it no longer needs, along with its signing secret.

## Known gaps

- Destinations don't receive events yet: they can be subscribed to topics, but
  delivering to them isn't built.
- Webhooks are the only kind of destination. Amazon SQS, RabbitMQ, and Kafka
  destinations are planned once webhook delivery is solid.
- Signed webhooks use a shared secret only. Bearer tokens and API keys aren't
  supported.
- A destination's URL and signing settings can't be changed after it's
  registered: delete it and register it again.
- No sign-in or permissions: anyone who can reach Relay can see, change, and
  delete every destination. Run Relay where only trusted services and people can
  reach it.

## Rules

- [Routing](../domain/rules/routing.md)
- [Subscriptions](../domain/rules/subscriptions.md)

## Concepts

- [Destination](../domain/concepts/destination.md)
- [Subscription](../domain/concepts/subscription.md)
