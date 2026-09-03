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

	// RoutingKey is the opaque string used to decide which Destinations should
	// receive this Envelope. Its meaning is defined by the routing rules, not
	// by relay itself.
	RoutingKey string

	// Message is the original message body, preserved verbatim as JSON so that
	// it can be delivered downstream without loss of fidelity.
	Message json.RawMessage

	// CreatedAt is the moment the Envelope was accepted into relay.
	CreatedAt time.Time
}
