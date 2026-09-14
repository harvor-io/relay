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

	// Slug is the URL-safe identifier for the Source, unique across all
	// Sources. It is derived from Name at creation time when one is not
	// supplied explicitly, and is stable thereafter.
	Slug string

	// Description is an optional free-form explanation of what the Source is
	// and why it exists. It is nil when no description has been provided.
	Description *string

	// IsActive reports whether the Source currently accepts ingested events.
	// Deactivating a Source without deleting it lets ingestion be paused
	// while keeping its history, keys, and configuration intact.
	IsActive bool

	// CreatedAt is the moment the Source was first registered.
	CreatedAt time.Time

	// UpdatedAt is the moment the Source's details were last modified. It
	// equals CreatedAt for a Source that has never been changed.
	UpdatedAt time.Time
}

// SourceKey is a credential used to verify HMAC-signed requests from a
// Source. A Source may have many keys, for example to support rotation
// without downtime.
type SourceKey struct {
	// ID is the public identifier of the key. It is a UUIDv7, so the value
	// is also roughly time-ordered by creation. Unlike SecretID, it is safe
	// to expose to clients and does not grant access to the key material.
	ID uuid.UUID

	// SourceID is the Source this key belongs to.
	SourceID uuid.UUID

	// SecretID references the key material held in the secrets store. It is
	// never exposed to clients.
	SecretID uuid.UUID

	// Name is a human-readable label used to recognise the key in logs and
	// dashboards, for example to distinguish it from other keys during
	// rotation.
	Name string

	// IsActive indicates whether the key is currently accepted for
	// verifying signed requests. Deactivating a key without deleting it
	// supports rotation without losing its history.
	IsActive bool

	// CreatedAt is the moment the key was created.
	CreatedAt time.Time

	// UpdatedAt is the moment the key's details were last modified. It
	// equals CreatedAt for a key that has never been changed.
	UpdatedAt time.Time
}
