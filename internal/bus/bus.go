// Package bus defines the abstraction Relay uses for its eventbus: the
// messaging infrastructure (e.g. RabbitMQ, Kafka) that backs the routing of
// events between Relay and its consumers. The Bus interface abstracts away
// the details of whichever specific broker is configured, so the rest of
// Relay depends on this interface rather than a concrete broker client.
// Concrete implementations live in sub-packages, e.g. internal/bus/rabbitmq.
package bus

import (
	"context"

	"github.com/harvor-io/relay/internal/models"
)

// Bus is the interface Relay uses to talk to the underlying eventbus,
// regardless of which broker backs it.
type Bus interface {
	// Setup idempotently provisions whatever the bus needs to operate, such
	// as exchanges, topics, or queues. It is safe to call on every startup:
	// implementations must ensure the required resources exist without
	// erroring or duplicating them if they are already present.
	Setup(ctx context.Context) error

	// Publish sends envelope to the bus, routed by its Topic so that
	// consumers bound to that topic receive it.
	Publish(ctx context.Context, envelope *models.Envelope) error

	// Apply idempotently provisions the topology destination needs to
	// receive deliveries: a delivery queue, a retry queue, and a dead-letter
	// queue, bound to receive envelopes matching destination.Subscriptions.
	// Implementations wire these together with a good default configuration
	// for moving messages between them (e.g. retrying a failed delivery
	// after a delay, and giving up into the dead-letter queue), so callers
	// do not have to manage that movement themselves. It is safe to call
	// multiple times for the same destination, but it only adds bindings
	// for the subscriptions currently in destination.Subscriptions — it does
	// not remove ones left over from a previous call whose Subscriptions no
	// longer include them. A caller that has changed a destination's
	// Subscriptions must call Remove and then Apply to rebind it from
	// scratch, rather than relying on Apply alone to reconcile the change.
	Apply(ctx context.Context, destination *models.Destination) error

	// Remove idempotently tears down the topology Apply provisioned for
	// destination. It is safe to call multiple times, including for a
	// destination whose topology was never created or already removed.
	Remove(ctx context.Context, destination *models.Destination) error
}
