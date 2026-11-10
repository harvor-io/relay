# Connect a producer

A producer team gets one of its services publishing events to Relay, and keeps
its keys fresh afterwards.

- **People:** Producer team, producer service
- **Starts when:** a service has events other teams should be able to receive
- **Done when:** the service's signed events are accepted under its source

## Steps

1. **Producer team** registers a source for the service, choosing its slug or
   letting Relay make one up from the name.
2. **Producer team** issues a key for the source and copies its secret, which
   Relay won't show again.
3. **Producer team** gives the secret to the service through its own secret
   handling.
4. **Producer service** publishes each event to the source, with its event type,
   signed with the key's secret.
5. **Relay** accepts the event and files it under the source's topic for that
   event type.

## Variations

- **Rotating a key:** the producer team issues a second key, moves the service
  onto it, then deactivates or deletes the first. Events signed with either are
  accepted while both are active.
- **Pausing a producer:** the producer team deactivates the source. Its events
  are refused until it's reactivated, and its keys are kept.
- **A retry after an error:** the producer service sends the event again with
  the same event ID, so it can't be recorded twice.

## Capabilities

- [Sources](../capabilities/sources.md)
- [Publishing events](../capabilities/publishing-events.md)
