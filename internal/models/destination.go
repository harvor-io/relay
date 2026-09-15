package models

import (
	"encoding/json"
	"time"

	"github.com/gofrs/uuid/v5"
)

// DestinationType identifies the delivery mechanism a Destination uses.
type DestinationType string

// DestinationTypeWebhook delivers data by making an outbound HTTP request to
// a configured URL. It is currently the only supported DestinationType.
const DestinationTypeWebhook DestinationType = "webhook"

// Destination represents a target that relay forwards ingested data to, such as
// a downstream service, a database sink, or an outbound webhook consumer. Data
// that arrives from a Source is delivered to one or more Destinations, and each
// delivery can be traced back to the Destination it was sent to.
type Destination struct {
	// ID is the stable, globally unique identifier for the Destination. It is
	// a UUIDv7, so the value is also roughly time-ordered by creation.
	ID uuid.UUID

	// Name is the human-readable label used to recognise the Destination in
	// logs, dashboards, and configuration.
	Name string

	// Type is the delivery mechanism this Destination uses. It is fixed at
	// creation time.
	Type DestinationType

	// Config is the Type-specific delivery configuration, stored verbatim as
	// JSON and fixed at creation time. For DestinationTypeWebhook, it decodes
	// as WebhookConfig.
	Config json.RawMessage

	// Description is an optional free-form explanation of what the Destination
	// is and why it exists. It is nil when no description has been provided.
	Description *string

	// Subscriptions lists the topic patterns this Destination wants to
	// receive Envelopes for. An Envelope is delivered here if its Topic
	// matches at least one Subscription.
	Subscriptions []Subscription

	// IsActive reports whether the Destination currently accepts deliveries.
	// When false, data is not forwarded here even though the Destination
	// remains registered.
	IsActive bool

	// CreatedAt is the moment the Destination was first registered.
	CreatedAt time.Time

	// UpdatedAt is the moment the Destination's details were last modified. It
	// equals CreatedAt for a Destination that has never been changed.
	UpdatedAt time.Time
}
