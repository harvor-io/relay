// Package secrets defines the persistence boundary for storing secret values,
// such as API keys and tokens used to authenticate with Sources and
// Destinations. Concrete implementations live in sub-packages (currently only
// sqlite).
package secrets

import (
	"context"
	"errors"

	"github.com/gofrs/uuid/v5"
)

// ErrNotFound is returned by a Store when a lookup or delete targets a Secret
// that does not exist.
var ErrNotFound = errors.New("secrets: record not found")

// ErrConflict is returned by a Store when a write would violate a uniqueness
// constraint, such as creating a Secret whose ID is already taken.
var ErrConflict = errors.New("secrets: conflicting record")

// Secret is a stored secret value, such as an API key or token, referenced by
// ID.
type Secret struct {
	// ID is the stable, globally unique identifier for the Secret.
	ID uuid.UUID

	// Value is the sensitive value itself.
	Value []byte
}

// Store persists and retrieves Secrets.
type Store interface {
	// Create persists a new Secret. Implementations assign the ID if it is
	// not already set, mutating the passed value. It returns ErrConflict if
	// the Secret's ID is already taken.
	Create(ctx context.Context, secret *Secret) error

	// Get returns the Secret with the given ID, or ErrNotFound.
	Get(ctx context.Context, id uuid.UUID) (*Secret, error)

	// Delete removes the Secret with the given ID, returning ErrNotFound if
	// it does not exist.
	Delete(ctx context.Context, id uuid.UUID) error
}
