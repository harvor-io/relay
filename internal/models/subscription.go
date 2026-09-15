package models

import "strings"

// Subscription is a value object describing a topic pattern a Destination
// wants to receive Envelopes for. It is either an exact Topic, or a prefix
// ending in ".*", which matches any Topic falling under that prefix (for
// example, "crm.*" matches "crm.account.created").
type Subscription string

// String returns the Subscription's string representation.
func (s Subscription) String() string {
	return string(s)
}

// Matches reports whether topic satisfies this Subscription: either an exact
// match, or, when the Subscription ends in ".*", a Topic starting with the
// prefix that precedes it.
func (s Subscription) Matches(topic Topic) bool {
	if prefix, ok := strings.CutSuffix(string(s), "*"); ok {
		return strings.HasPrefix(topic.String(), prefix)
	}
	return string(s) == topic.String()
}
