# Tomás Reyes, consumer engineer

## Who

A backend engineer on the analytics team at a mid-size company, whose service
needs to know when orders and customers change. Uses Relay to have those events
delivered to a webhook, without asking each producing team to build an
integration.

## Knows

Comfortable with REST APIs, JSON, and signing requests with HMAC, and has
verified signed webhooks from third-party services. Has consumed from SQS or
RabbitMQ in application code, but has never run a broker. New to Relay.

## Context

Works on a laptop and touches Relay rarely: when their service needs a new kind
of event, or its endpoint changes. Works from the API docs, curl, and their own
service's code. Relay is a side task in the middle of feature work.

## Expects

Thinks of Relay like a managed broker such as SNS and SQS, or EventBridge:
subscribe to the topics they want and receive what matches, with retries,
timeouts, and dead letters when their endpoint is down. Expects to choose what
they receive without talking to the producing team.

## Cares about

Trusts Relay when they can tell whether an event was delivered, and when docs
match what happens. Gives up when events stop arriving with no way to tell
whether the producer stopped sending or Relay stopped delivering, or when a
delivery's signature can't be verified with what Relay told them.

## Based on

The user's answers, and the consumer team in the product docs. Assumed: the
company and the analytics team, and that they receive events by webhook, the
only kind of destination Relay has.
