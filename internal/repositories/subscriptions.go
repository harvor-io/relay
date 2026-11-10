package repositories

import (
	"context"

	"github.com/exalynt/turn/paginator/cursor"
	"github.com/gofrs/uuid/v5"

	"github.com/harvor-io/relay/internal/models"
)

// SubscriptionRepository stores and retrieves models.Subscription records.
type SubscriptionRepository interface {
	// Create persists a new Subscription. Implementations assign the ID and
	// timestamps if they are not already set, mutating the passed value. It
	// returns ErrConflict if the Destination already has a Subscription to
	// the Source.
	Create(ctx context.Context, subscription *models.Subscription) error

	// Get returns the Subscription with the given ID, or ErrNotFound.
	Get(ctx context.Context, id uuid.UUID) (*models.Subscription, error)

	// List returns the page of a Destination's Subscriptions query selects,
	// ordered by Source slug, then ID. A selector turn rejects is returned
	// as its paginator error.
	List(ctx context.Context, query SubscriptionQuery) (cursor.Page[models.Subscription], error)

	// ListByDestination returns every Subscription belonging to
	// destinationID, unpaginated, for Relay's own use, such as restoring a
	// Destination's routing when it is activated.
	ListByDestination(ctx context.Context, destinationID uuid.UUID) ([]models.Subscription, error)

	// Update overwrites the Topics of an existing Subscription and refreshes
	// its UpdatedAt timestamp. It returns ErrNotFound if no such
	// Subscription exists.
	Update(ctx context.Context, subscription *models.Subscription) error

	// Delete removes the Subscription with the given ID, returning
	// ErrNotFound if it does not exist.
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteByDestination removes every Subscription belonging to
	// destinationID. Having none is not an error.
	DeleteByDestination(ctx context.Context, destinationID uuid.UUID) error
}

// SubscriptionQuery selects a page of one Destination's Subscriptions.
type SubscriptionQuery struct {
	DestinationID uuid.UUID
	Page          cursor.Selector
}
