# Event

One message a [source](source.md) publishes to Relay, such as an order being
created. Once Relay accepts it, it's also called an envelope.

## Details

An event has an ID, the source it came from, an event type, and its data, kept
exactly as the producer sent it. The event type says what kind of event it is,
and by convention is `<resource>.<action>`, such as `order.created`. Its source
and event type together make its [topic](topic.md).

The producer can choose the event's ID. Otherwise Relay gives it one.
