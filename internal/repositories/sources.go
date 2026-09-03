package repositories

import (
	"context"

	"github.com/gofrs/uuid/v5"

	"github.com/harvor-io/relay/internal/models"
)

// SourceRepository stores and retrieves models.Source records: the origins that
// relay ingests data from.
type SourceRepository interface {
	// Create persists a new Source. Implementations assign the ID and
	// timestamps if they are not already set, mutating the passed value.
	Create(ctx context.Context, source *models.Source) error

	// Get returns the Source with the given ID, or ErrNotFound.
	Get(ctx context.Context, id uuid.UUID) (*models.Source, error)

	// List returns every Source, ordered by name.
	List(ctx context.Context) ([]models.Source, error)

	// Update overwrites the mutable fields of an existing Source and refreshes
	// its UpdatedAt timestamp. It returns ErrNotFound if no such Source exists.
	Update(ctx context.Context, source *models.Source) error

	// Delete removes the Source with the given ID, returning ErrNotFound if it
	// does not exist.
	Delete(ctx context.Context, id uuid.UUID) error
}
