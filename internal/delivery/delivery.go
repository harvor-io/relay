// Package delivery defines how Relay hands an envelope to a destination,
// whatever kind of destination it is. Each models.DestinationType has its own
// Deliverer, and concrete implementations live in sub-packages, e.g.
// internal/delivery/webhook.
package delivery

import (
	"context"

	"github.com/harvor-io/relay/internal/models"
)

// Deliverer delivers envelopes to destinations of one models.DestinationType.
// It reads the destination's config on every call, so one Deliverer serves
// every destination of its type.
type Deliverer interface {
	// Deliver sends envelope to destination. A nil error means the
	// destination accepted it. Any error means it may not have, and the
	// envelope should be delivered again, so a destination can receive the
	// same envelope more than once and should deduplicate by its ID.
	Deliver(ctx context.Context, destination *models.Destination, envelope *models.Envelope) error
}
