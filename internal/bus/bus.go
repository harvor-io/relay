// Package bus defines the abstraction Relay uses for its eventbus: the
// messaging infrastructure (e.g. RabbitMQ, Kafka) that backs the routing of
// events between Relay and its consumers. The Bus interface abstracts away
// the details of whichever specific broker is configured, so the rest of
// Relay depends on this interface rather than a concrete broker client.
// Concrete implementations live in sub-packages, e.g. internal/bus/rabbitmq.
package bus

import "context"

// Bus is the interface Relay uses to talk to the underlying eventbus,
// regardless of which broker backs it.
type Bus interface {
	// Setup idempotently provisions whatever the bus needs to operate, such
	// as exchanges, topics, or queues. It is safe to call on every startup:
	// implementations must ensure the required resources exist without
	// erroring or duplicating them if they are already present.
	Setup(ctx context.Context) error
}
