package repositories

import (
	"context"

	"github.com/gofrs/uuid/v5"

	"github.com/harvor-io/relay/internal/models"
)

// EnvelopeRepository stores and retrieves models.Envelope records: the units of
// data moving through relay. Envelopes are immutable once written, so there is
// no update or delete.
type EnvelopeRepository interface {
	// Create persists a new Envelope. Implementations assign the ID and
	// CreatedAt timestamp if they are not already set, mutating the passed
	// value.
	Create(ctx context.Context, envelope *models.Envelope) error

	// Get returns the Envelope with the given ID, or ErrNotFound.
	Get(ctx context.Context, id uuid.UUID) (*models.Envelope, error)

	// List returns every Envelope, newest first.
	List(ctx context.Context) ([]models.Envelope, error)
}
