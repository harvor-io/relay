# Subscription

The link that says a [destination](destination.md) wants the events of a
[topic](topic.md), or of every topic a pattern matches.

## Details

A destination's owner creates its subscriptions, and the source's owner doesn't
change anything. A subscription names a topic exactly, such as
`orders.order.created`, or by a wildcard pattern that also matches event types
the source starts publishing later. A subscription names no
[source](source.md): the topic already starts with the source's slug, so a
destination can subscribe before the source exists, and its subscriptions stay
if the source is deleted.

One event can go to every destination subscribed to its topic, and a destination
can start or stop receiving events without the source knowing.
