package models

import (
	"time"

	"github.com/gofrs/uuid/v5"
)

// Subscription says that a Destination wants some of one Source's events. A
// Destination has at most one Subscription per Source, and every
// Subscription names at least one Topic.
type Subscription struct {
	// ID is the stable, globally unique identifier for the Subscription. It
	// is a UUIDv7, so the value is also roughly time-ordered by creation.
	ID uuid.UUID

	// DestinationID is the Destination that receives the matching events.
	DestinationID uuid.UUID

	// SourceID is the Source whose events the Subscription covers. The
	// Subscription is kept if the Source is deleted.
	SourceID uuid.UUID

	// Source is the slug of the Source, captured when the Subscription was
	// created. Every Topic starts with it.
	Source string

	// Topics are the topics the Destination wants, de-duplicated and
	// sorted. Each is either an exact topic, such as "orders.order.created",
	// or a pattern ending in ".*", such as "orders.order.*", which matches
	// every topic that starts with what comes before the "*" and has at
	// least one more part.
	Topics []Topic

	// CreatedAt is the moment the Subscription was created.
	CreatedAt time.Time

	// UpdatedAt is the moment the Subscription's topics last changed. It
	// equals CreatedAt for a Subscription that has never been changed.
	UpdatedAt time.Time
}
