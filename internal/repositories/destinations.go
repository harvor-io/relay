package repositories

import (
	"context"

	"github.com/exalynt/turn/paginator/cursor"
	"github.com/gofrs/uuid/v5"

	"github.com/harvor-io/relay/internal/models"
)

// DestinationRepository stores and retrieves models.Destination records: the
// targets that relay forwards ingested data to.
type DestinationRepository interface {
	// Create persists a new Destination. Implementations assign the ID and
	// timestamps if they are not already set, mutating the passed value. It
	// returns ErrConflict if the Destination's ID is already taken.
	Create(ctx context.Context, destination *models.Destination) error

	// Get returns the Destination with the given ID, or ErrNotFound.
	Get(ctx context.Context, id uuid.UUID) (*models.Destination, error)

	// List returns the page of Destinations query selects, ordered by name,
	// then ID. A selector turn rejects is returned as its paginator error.
	List(ctx context.Context, query DestinationQuery) (cursor.Page[models.Destination], error)

	// Update overwrites the mutable fields of an existing Destination and
	// refreshes its UpdatedAt timestamp. It returns ErrNotFound if no such
	// Destination exists.
	Update(ctx context.Context, destination *models.Destination) error

	// Delete removes the Destination with the given ID, returning
	// ErrNotFound if it does not exist.
	Delete(ctx context.Context, id uuid.UUID) error
}

// DestinationQuery selects a page of Destinations. Destinations have no
// filters yet.
type DestinationQuery struct {
	Page cursor.Selector
}
