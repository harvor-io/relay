# Eventbus

The message broker Relay publishes accepted envelopes to, so that routing them
to destinations is done by the broker's own primitives rather than by Relay.

**Code:** `internal/bus` (the `Bus` interface), `internal/bus/rabbitmq`,
`sourceService.CreateEvent` in `internal/services/sources.go`,
`destinationService` in `destinations.go`, and `subscriptionService` in
`subscriptions.go`. Wired up in `internal/commands/serve.go`.

## How it works

On startup, `relay serve` dials RabbitMQ and calls `Bus.Setup`, which declares
one durable topic exchange, `relay_ingest`. Setup is idempotent and runs every
start.

When a source publishes an event, `CreateEvent` writes the envelope to the
database first, then calls `Bus.Publish`. The message's routing key is the
envelope's [topic](../../../product/domain/concepts/topic.md),
`<source slug>.<event type>`, and its body is `envelopeMessage`, an explicit
projection of `models.Envelope`.

Each destination has a durable queue, `relay.destination.<id>`, declared when
the destination is created. Each of an active destination's
[subscriptions](../../../product/domain/concepts/subscription.md) is one
binding of that queue on `relay_ingest`, keyed by its topic. Neither a
subscription nor a binding names a source: the topic carries the source's
slug, and the broker decides which queues get a message. Relay never matches
sources to destinations itself. The services keep the bindings in step with
what's stored:

- Subscribing, replacing a destination's subscriptions, and unsubscribing bind
  and unbind them, but only while the destination is active.
- Deactivating a destination unbinds all of its subscriptions, and activating
  it binds them again from the database.
- Deleting a destination deletes its queue.

`Bus` speaks in destinations and subscriptions, and the driver maps them onto
queues and bindings. Every method is idempotent, which the interface's doc
comment makes the contract for every driver. The services write the database
first and call the bus after, so a retried or repeated call is always safe.

The target shape still to come: a delivery worker per destination consumes its
queue and delivers by the destination's type. Nothing consumes the queues yet.

## Tradeoffs

- **Routing done by the broker, not by Relay:** fan-out and topic matching
  already exist in RabbitMQ, Kafka, and the like, so Relay doesn't reimplement
  them. **Costs:** Relay can't accept events without a reachable broker, and
  every eventbus driver has to map subscriptions onto its own primitives.

## Gotchas

- The broker is an internal detail and must stay invisible: never put exchange,
  queue, or broker names or settings in an API resource, error message, or
  destination config. See
  [The message broker stays hidden](../../../product/domain/rules/routing.md#the-message-broker-stays-hidden).
- RabbitMQ as the eventbus is not the same thing as a RabbitMQ destination type.
  Don't reuse the eventbus connection or exchange to implement that destination.
- The envelope write and the publish aren't atomic. If the publish fails, the
  envelope is already in the database and the request returns 500 (the TODO in
  `CreateEvent`). Fixing this probably means an outbox, which would also change
  how ingestion latency behaves.
- Messages are published with `mandatory` false, so RabbitMQ silently drops an
  event no queue's binding matches. That's how an inactive destination misses
  events, as the
  [routing rules](../../../product/domain/rules/routing.md#an-inactive-destination-misses-events)
  require.
- A subscription's trailing `*` binds as `#`, and `*` alone binds as `#` too.
  `#` matches zero or more parts, where the rule says one or more. So
  `orders.order.*` also matches an event whose topic is exactly
  `orders.order`. Only an event type equal to the pattern's prefix can tell
  the difference.
- A queue gets one copy of a message however many of its bindings match it,
  so overlapping subscriptions, such as `orders.*` and `orders.order.*`,
  don't duplicate events. For the same reason, unsubscribing from a topic
  doesn't stop its events while another of the destination's patterns still
  matches them.
- The database write and the bus call aren't atomic. If the bus call fails,
  the request returns 500 with the database already changed, and the bindings
  no longer match what's stored. Repeating the request repairs it: subscribing
  and replacing rebind every topic they keep, and unsubscribing unbinds even
  when nothing was stored. So does deactivating and then activating the
  destination, which rebinds it from the database, though that can't remove a
  binding the database no longer knows about.
- `destinationService.Delete` deletes the destination's row and subscriptions
  before its queue. If deleting the queue fails, the queue and any events in
  it are left on the broker, and Relay can't reach them anymore: delete
  `relay.destination.<id>` on the broker by hand.
- A destination can exist without a queue: one created before queues existed,
  or one whose queue declaration failed after its row was written. It gets one
  on its first subscription, because `Subscribe` declares the queue before
  binding.
- The AMQP connection is opened once at startup and never re-established. If it
  drops, every publish fails until the process restarts. `/readyz` reports the
  `eventbus` check down, so the process leaves rotation, but `/livez` stays up,
  so a liveness probe won't restart it: someone has to.
- `relay serve` fails to start if RabbitMQ isn't reachable, so the end-to-end
  tests need it running. The acceptance tests don't: they use the generated
  `bus.BusMock` instead.
- The message body is a wire format that consumers will depend on. Change
  `envelopeMessage` deliberately, not as a side effect of changing
  `models.Envelope`.
- `Publish` opens a new channel per message. Fine at current volume; the first
  place to look if publish throughput becomes a problem.

## Known gaps

- **Queues have no cap, and nothing consumes them:** every event an active
  destination subscribes to piles up in its queue, using broker storage, until
  someone deletes the destination or purges the queue by hand. A cap added
  later means deleting each queue and declaring it again, because RabbitMQ
  refuses to redeclare a queue with different arguments. **Until:** delivery
  is built.

- **RabbitMQ is the only driver:** `EVENTBUS_DRIVER` accepts only `rabbitmq`, and
  serve refuses anything else. **Until:** not decided.

## Changing it

**Changing the `Bus` interface:** keep every method idempotent, then run
`go generate ./internal/bus/` to regenerate `BusMock`, or the acceptance tests
stop compiling.

**Adding an eventbus driver:** implement `bus.Bus` in a new package under
`internal/bus/`, with every method idempotent, including `Setup`. Map
destinations and subscriptions onto the broker's own primitives, translating a
`*` into its wildcard, and select it by `EVENTBUS_DRIVER`
in `serve.go` and `internal/config`. Keep the routing key as the envelope's topic
and the body as the same wire format, or consumers break when the driver
changes.
