# Source key

A credential a [source](source.md)'s services sign [events](event.md) with, so
Relay knows an event really comes from that source.

## Details

A key belongs to one source and has a name and a secret. The secret is shown
once, when the key is issued, and Relay never shows it again. A source can have
several keys at once, which is how keys are rotated.

## Lifecycle

Active ⇄ Inactive → Deleted

A key starts active. Only active keys are accepted.

## Not to be confused with

- A destination's signing secret: Relay uses that to sign what it delivers to a
  [destination](destination.md), rather than to check what a source sends.
