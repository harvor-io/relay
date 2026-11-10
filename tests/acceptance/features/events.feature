Feature: Events
  A source's services publish events to Relay, each signed with one of the
  source's keys.

  Background:
    Given the "orders" source has an active key "primary"

  Rule: Only active sources accept events

    Example: An active source accepts an event
      When the orders service publishes an "order.created" event signed with "primary"
      Then the event is accepted under the topic "orders.order.created"

    Example: A deactivated source refuses events
      Given the "orders" source is deactivated
      When the orders service publishes an "order.created" event signed with "primary"
      Then the event is refused

  Rule: Events are signed by an active key

    Example: Any active key is accepted, so keys can rotate
      Given the "orders" source also has an active key "secondary"
      When the orders service publishes an "order.created" event signed with "secondary"
      Then the event is accepted

    Example: A deactivated key is refused
      Given the "primary" key is deactivated
      When the orders service publishes an "order.created" event signed with "primary"
      Then the event is refused

    Example: Another source's key is refused
      Given the "billing" source has an active key "ledger"
      When the orders service publishes an "order.created" event signed with "ledger"
      Then the event is refused

  Rule: Accepted events never change

    Example: An accepted event's ID can't be reused
      Given the orders service has published an "order.created" event
      When the orders service publishes an "order.updated" event with the same ID
      Then the event is refused
      And the accepted event is still an "order.created" event
