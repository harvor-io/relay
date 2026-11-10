# Subscription

The link that says a [destination](destination.md) wants a
[source](source.md)'s events, for one or more of its [topics](topic.md).

## Details

A destination's owner creates its subscriptions, and the source's owner doesn't
change anything. A subscription names topics exactly, such as
`orders.order.created`, or by a wildcard pattern that also matches event types
the source starts publishing later. A subscription covers one source, so a
destination that wants events from several sources has a subscription for each.

One event can go to every destination subscribed to its topic, and a destination
can start or stop receiving events without the source knowing.
