package models

import (
	"time"

	"github.com/gofrs/uuid/v5"
)

// Subscription is the link that says a Destination wants the events of one
// Topic, or of every Topic a pattern matches. It has no ID of its own: the
// Destination and the Topic identify it. A Subscription names no Source:
// the Topic already starts with the Source's slug, and events are routed by
// Topic alone, so a Destination can subscribe before the Source exists and
// keeps its Subscriptions if the Source is deleted.
type Subscription struct {
	// DestinationID is the Destination that wants the matching events.
	DestinationID uuid.UUID

	// Topic is what the Destination subscribes to: an exact topic, such as
	// "orders.order.created", or a pattern ending in "*", such as
	// "orders.order.*" or "orders.*", which matches every topic that starts
	// with what comes before the "*" and has at least one more part. "*"
	// alone matches every topic.
	Topic Topic

	// CreatedAt is the moment the Destination subscribed.
	CreatedAt time.Time
}
