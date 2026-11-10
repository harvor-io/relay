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

//go:generate moq -out bus_mock.go . Bus

// Bus is the interface Relay uses to talk to the underlying eventbus,
// regardless of which broker backs it. It speaks in Relay's domain: each
// implementation maps destinations and subscriptions onto its broker's own
// primitives, and none of those primitives leak out of it.
//
// Every method is idempotent: calling it again with the same input succeeds
// and leaves the bus as the first call did. Relay calls the bus after the
// database write it follows, so a failed call can be retried, and an
// operation can be repeated to repair the bus, without either making things
// worse.
type Bus interface {
	// Setup idempotently provisions whatever the bus needs to operate, such
	// as exchanges, topics, or queues. It is safe to call on every startup:
	// implementations must ensure the required resources exist without
	// erroring or duplicating them if they are already present.
	Setup(ctx context.Context) error

	// Publish sends envelope to the bus, routed by its Topic so that
	// consumers bound to that topic receive it.
	Publish(ctx context.Context, envelope *models.Envelope) error

	// AddDestination makes sure events can be routed to destination.
	AddDestination(ctx context.Context, destination *models.Destination) error

	// RemoveDestination removes destination and stops every event routed
	// to it, along with any events waiting for it. A destination the bus
	// doesn't have is not an error.
	RemoveDestination(ctx context.Context, destination *models.Destination) error

	// Subscribe routes events matching each of subscription's Topics to its
	// destination, adding the destination first if the bus doesn't have it.
	Subscribe(ctx context.Context, subscription *models.Subscription) error

	// Unsubscribe stops routing events matching subscription's Topics to its
	// destination. Topics that aren't routed, or a destination the bus
	// doesn't have, are not an error.
	Unsubscribe(ctx context.Context, subscription *models.Subscription) error

	// ChangeSubscription moves a subscription from previous's Topics to
	// current's: it routes every Topic in current, then stops routing the
	// Topics only previous has, so a Topic in both is never interrupted.
	ChangeSubscription(ctx context.Context, previous, current *models.Subscription) error
}
