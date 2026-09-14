package models

import (
	"encoding/json"
	"time"

	"github.com/gofrs/uuid/v5"
)

// Envelope represents a single unit of data as it travels through relay: the
// raw message received from a Source, wrapped with the metadata needed to route
// and deliver it to Destinations. An Envelope is immutable once created; its
// delivery attempts and outcomes are tracked separately.
type Envelope struct {
	// ID is the stable, globally unique identifier for the Envelope. It is a
	// UUIDv7, so the value is also roughly time-ordered by creation.
	ID uuid.UUID

	// SourceID is the identifier of the Source that produced this Envelope.
	SourceID uuid.UUID

	// Source is the slug of the originating Source, captured at the time the
	// Envelope was created so it remains stable even if the Source is later
	// renamed.
	Source string

	// Type identifies the kind of message this Envelope carries within its
	// Source, e.g. "created" or "updated". Combined with Source, it forms the
	// Envelope's Topic.
	Type string

	// Message is the original message body, preserved verbatim as JSON so that
	// it can be delivered downstream without loss of fidelity.
	Message json.RawMessage

	// CreatedAt is the moment the Envelope was accepted into relay.
	CreatedAt time.Time
}

// Topic returns the value object used to decide which Destinations should
// receive this Envelope, derived from its Source and Type.
func (e Envelope) Topic() Topic {
	return NewTopic(e.Source, e.Type)
}
