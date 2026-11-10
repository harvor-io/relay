package repositories

import (
	"context"

	"github.com/exalynt/turn/paginator/cursor"
	"github.com/gofrs/uuid/v5"

	"github.com/harvor-io/relay/internal/models"
)

// SubscriptionRepository stores and retrieves models.Subscription records,
// each identified by its Destination and Topic.
type SubscriptionRepository interface {
	// Create persists a new Subscription, assigning its CreatedAt and
	// mutating the passed value. It returns ErrConflict if the Destination
	// is already subscribed to the Topic.
	Create(ctx context.Context, subscription *models.Subscription) error

	// Get returns destinationID's Subscription to topic, or ErrNotFound.
	Get(ctx context.Context, destinationID uuid.UUID, topic models.Topic) (*models.Subscription, error)

	// List returns the page of a Destination's Subscriptions query selects,
	// ordered by Topic. A selector turn rejects is returned as its paginator
	// error.
	List(ctx context.Context, query SubscriptionQuery) (cursor.Page[models.Subscription], error)

	// ListByDestination returns every Subscription belonging to
	// destinationID, unpaginated, for Relay's own use, such as restoring a
	// Destination's routing when it is activated.
	ListByDestination(ctx context.Context, destinationID uuid.UUID) ([]models.Subscription, error)

	// Replace makes destinationID's Subscriptions exactly topics, in one
	// transaction: it removes the ones topics doesn't name and adds the
	// ones it does, keeping the CreatedAt of those it already had.
	Replace(ctx context.Context, destinationID uuid.UUID, topics []models.Topic) error

	// Delete removes destinationID's Subscription to topic, returning
	// ErrNotFound if there is none.
	Delete(ctx context.Context, destinationID uuid.UUID, topic models.Topic) error

	// DeleteByDestination removes every Subscription belonging to
	// destinationID. Having none is not an error.
	DeleteByDestination(ctx context.Context, destinationID uuid.UUID) error
}

// SubscriptionQuery selects a page of one Destination's Subscriptions.
type SubscriptionQuery struct {
	DestinationID uuid.UUID
	Page          cursor.Selector
}
