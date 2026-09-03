package repositories

import (
	"context"

	"github.com/gofrs/uuid/v5"

	"github.com/harvor-io/relay/internal/models"
)

// DestinationRepository stores and retrieves models.Destination records: the
// targets that relay forwards data to.
type DestinationRepository interface {
	// Create persists a new Destination. Implementations assign the ID and
	// timestamps if they are not already set, mutating the passed value.
	Create(ctx context.Context, destination *models.Destination) error

	// Get returns the Destination with the given ID, or ErrNotFound.
	Get(ctx context.Context, id uuid.UUID) (*models.Destination, error)

	// List returns every Destination, ordered by name.
	List(ctx context.Context) ([]models.Destination, error)

	// Update overwrites the mutable fields of an existing Destination
	// (including IsActive) and refreshes its UpdatedAt timestamp. It returns
	// ErrNotFound if no such Destination exists.
	Update(ctx context.Context, destination *models.Destination) error

	// Delete removes the Destination with the given ID, returning ErrNotFound
	// if it does not exist.
	Delete(ctx context.Context, id uuid.UUID) error
}
