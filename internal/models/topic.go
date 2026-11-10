package models

import "fmt"

// Topic is a value object that identifies what an Envelope is about, so that
// the eventbus can decide which Destinations should receive it. It is
// derived from the originating Source's slug and the Envelope's Type,
// formatted as "<source>.<type>". A Subscription's Topic may instead be a
// pattern ending in ".*"; see Subscription.Topics.
type Topic string

// WildcardSuffix ends a Subscription's pattern Topic, such as
// "orders.order.*".
const WildcardSuffix = ".*"

// NewTopic builds the Topic for an Envelope from its Source slug and Type.
func NewTopic(source, typ string) Topic {
	return Topic(fmt.Sprintf("%s.%s", source, typ))
}

// String returns the Topic's string representation.
func (t Topic) String() string {
	return string(t)
}
