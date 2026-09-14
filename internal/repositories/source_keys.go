package repositories

import (
	"context"

	"github.com/gofrs/uuid/v5"

	"github.com/harvor-io/relay/internal/models"
)

// SourceKeyRepository stores and retrieves models.SourceKey records: the
// HMAC credentials used to verify signed requests from a Source.
type SourceKeyRepository interface {
	// Create persists a new SourceKey. Implementations assign the ID and
	// timestamps if they are not already set, mutating the passed value.
	Create(ctx context.Context, key *models.SourceKey) error

	// Get returns the SourceKey with the given ID, or ErrNotFound.
	Get(ctx context.Context, id string) (*models.SourceKey, error)

	// ListBySource returns every SourceKey belonging to sourceID, ordered
	// by name.
	ListBySource(ctx context.Context, sourceID uuid.UUID) ([]models.SourceKey, error)

	// Update overwrites the mutable fields of an existing SourceKey and
	// refreshes its UpdatedAt timestamp. It returns ErrNotFound if no such key
	// exists.
	Update(ctx context.Context, key *models.SourceKey) error

	// Delete removes the SourceKey with the given ID, returning
	// ErrNotFound if it does not exist.
	Delete(ctx context.Context, id string) error
}
