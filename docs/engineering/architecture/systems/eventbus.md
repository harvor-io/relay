# Eventbus

The message broker Relay publishes accepted envelopes to, so that routing them
to destinations is done by the broker's own primitives rather than by Relay.

**Code:** `internal/bus` (the `Bus` interface), `internal/bus/rabbitmq`, and
`sourceService.CreateEvent` in `internal/services/sources.go`. Wired up in
`internal/commands/serve.go`.

## How it works

On startup, `relay serve` dials RabbitMQ and calls `Bus.Setup`, which declares
one durable topic exchange, `relay_ingest`. Setup is idempotent and runs every
start.

When a source publishes an event, `CreateEvent` writes the envelope to the
database first, then calls `Bus.Publish`. The message's routing key is the
envelope's [topic](../../../product/domain/concepts/topic.md),
`<source slug>.<event type>`, and its body is `envelopeMessage`, an explicit
projection of `models.Envelope`.

The target shape: each destination's
[subscriptions](../../../product/domain/concepts/subscription.md) become queue
bindings on `relay_ingest`, and a delivery worker per destination consumes its
queue and delivers by the destination's type. The broker decides which bound
queues get a message. Relay never matches sources to destinations itself.
None of the consuming side exists yet.

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
- Messages are published with `mandatory` false, so while no queue is bound to
  `relay_ingest`, RabbitMQ silently drops every message. Today that's all of
  them: an accepted event exists only in the envelope store.
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

- **RabbitMQ is the only driver:** `EVENTBUS_DRIVER` accepts only `rabbitmq`, and
  serve refuses anything else. **Until:** not decided.

## Changing it

**Changing the `Bus` interface:** run `go generate ./internal/bus/` to
regenerate `BusMock`, or the acceptance tests stop compiling.

**Adding an eventbus driver:** implement `bus.Bus` in a new package under
`internal/bus/`, with an idempotent `Setup`, and select it by `EVENTBUS_DRIVER`
in `serve.go` and `internal/config`. Keep the routing key as the envelope's topic
and the body as the same wire format, or consumers break when the driver
changes.
