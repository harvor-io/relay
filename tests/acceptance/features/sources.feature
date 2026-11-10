Feature: Sources
  A source is a producer registered with Relay. Its slug is the first part of
  every topic it publishes to.

  Rule: A slug is unique across Relay

    Example: A slug in use can't be taken again
      Given the "orders" source is registered
      When a source is registered with the slug "orders"
      Then the source is refused

    Example: A derived slug steps aside for one in use
      Given the "orders" source is registered
      When a source named "Orders" is registered without a slug
      Then it is registered with the slug "orders-2"

  Rule: A slug never changes

    Example: Renaming a source keeps its slug
      Given the "orders" source is registered
      When the source is renamed "Order service"
      Then its slug is still "orders"
