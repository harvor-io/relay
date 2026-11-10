# Events rules

Which events Relay accepts from a source, and what happens to them afterwards.

## Only active sources accept events

A source accepts events only while it's active. Events published to a
deactivated source are refused.

- **Kind:** Core
- **Why:** so a producer can be paused without losing its keys or history.
- **Acceptance test:** [events.feature › Only active sources accept events](../../../../tests/acceptance/features/events.feature)

## Events are signed by an active key

An event is accepted only when it's signed with the secret of one of its
source's active keys. Any active key will do, so keys can be rotated without
downtime.

- **Kind:** Core
- **Acceptance test:** [events.feature › Events are signed by an active key](../../../../tests/acceptance/features/events.feature)

## Accepted events never change

Once Relay accepts an event, neither it nor anything about it changes.

- **Kind:** Core
- **Acceptance test:** None yet

## Events are kept only for a limited time

Relay isn't a permanent record of events. It keeps them long enough to retry
deliveries and see what happened, then removes them once they're older than a
configurable maximum age.

- **Kind:** Core
- **Why:** replay and archiving belong to a store built for them, not to Relay.
- **Acceptance test:** None yet
