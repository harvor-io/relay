// Package models contains the core domain entities that the app persists and
// operates on. These types are storage-agnostic: they describe the shape of a
// concept, not how it is stored or transported.
package models

import (
	"time"

	"github.com/gofrs/uuid/v5"
)

// Source represents an origin that relay ingests data from, such as an upstream
// feed, an external service, or a webhook producer. Each Source is registered
// once and then referenced by the records that flow through the system, so that
// every piece of data can be traced back to where it came from.
type Source struct {
	// ID is the stable, globally unique identifier for the Source. It is a
	// UUIDv7, so the value is also roughly time-ordered by creation.
	ID uuid.UUID

	// Name is the human-readable label used to recognise the Source in logs,
	// dashboards, and configuration.
	Name string

	// Description is an optional free-form explanation of what the Source is
	// and why it exists. It is nil when no description has been provided.
	Description *string

	// CreatedAt is the moment the Source was first registered.
	CreatedAt time.Time

	// UpdatedAt is the moment the Source's details were last modified. It
	// equals CreatedAt for a Source that has never been changed.
	UpdatedAt time.Time
}
