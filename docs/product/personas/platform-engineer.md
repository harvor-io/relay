# Dana Whitfield, platform engineer

## Who

The engineer at a small company who looks after shared infrastructure alongside
their own product work. Runs Relay so the other teams have one place to publish
and receive events.

## Knows

Comfortable with Docker Compose, services configured through environment
variables, and hosted databases, and has run RabbitMQ before. Knows Relay's
pieces from setting it up, but not every detail of its API. Reads Relay's code
only as a last resort.

## Context

Sets Relay up once with Docker Compose from a laptop, then checks on it now and
then. The person other teams come to when their events stop flowing, usually
while busy with something else.

## Expects

Thinks of Relay like running their own SNS and SQS. Expects one binary
configured by environment variables, migrations applied on start, a health check
for the load balancer, and logs that say what's wrong.

## Cares about

Trusts Relay when it's safe to run unattended: few moving parts, secrets
encrypted at rest, and it either recovers on its own or fails loudly. Gives up
when an outage can only be fixed by reading the code, or a secret leaks.

## Based on

The user's answers, the self-hosted goals in the repository's README, and how
Relay is set up in CONTRIBUTING.md. Assumed: one person runs Relay part-time,
deploying it with Docker Compose.
