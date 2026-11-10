# Topic

What an [event](event.md) is about, used to decide which
[destinations](destination.md) receive it. It's the source's slug and the event
type, as `<source>.<event type>`, such as `orders.order.created`.

## Details

Every event has exactly one topic. A [subscription](subscription.md) names the
topics a destination wants, either exactly or by a wildcard pattern.

## Not to be confused with

- Event type: just the second half of the topic, which means something only
  within its [source](source.md).
