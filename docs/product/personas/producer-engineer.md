# Maya Okafor, producer engineer

## Who

A backend engineer on a product team at a mid-size company, who owns the orders
service. Uses Relay so other teams can react to orders without the orders
service having to call each of them.

## Knows

Comfortable with REST APIs, JSON, and signing requests with HMAC. Has published
to SQS or RabbitMQ from application code, but has never run a broker. New to
Relay, and knows only that their company runs it.

## Context

Works on a laptop and touches Relay rarely: when a service starts publishing a
new kind of event, or a key needs rotating. Works from the API docs, curl, and
their own service's code. Relay is a side task in the middle of feature work.

## Expects

Thinks of Relay like a managed broker such as SNS and SQS, or EventBridge:
publish to a topic, everyone subscribed gets it, and retries and dead letters
happen out of sight. Expects words like topic and subscription to mean what they
mean there, and expects an error to say what went wrong.

## Cares about

Trusts Relay when it's predictable: clear errors, docs that match what happens,
and no event ever lost without a trace. Gives up when an event they published
disappears with nothing to show where it went, or a signature is refused with no
hint why.

## Based on

The user's answers, and the producer team in the product docs. Assumed: the
company, the orders service, and that sources are set up by hand rather than by
automation.
