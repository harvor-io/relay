# Delivery

Handing an envelope to a destination, by the destination's type. Envelopes
reach it from the destination's queue on the [eventbus](../systems/eventbus.md)
through `Bus.Consume`.

**Code:** `internal/delivery` (the `Deliverer` interface) and
`internal/delivery/webhook`.

## How it works

There's one `Deliverer` per destination type, and it reads the destination's
config on every call, so one value serves every destination of its type. The
webhook deliverer POSTs the envelope as JSON to the config's URL. With hmac
auth, it reads the signing secret from the secret store, decrypts it, fills
the signing template with a Unix-seconds timestamp and the exact body, and sets
the timestamp and signature headers. Anything but a 2xx is an error.

An error means "deliver it again": `Bus.Consume` requeues the envelope, so
delivery is at least once and consumers deduplicate by envelope ID.

## Gotchas

- The webhook body, `message` in `internal/delivery/webhook`, is the contract
  consumers depend on. It's deliberately separate from the eventbus's
  `envelopeMessage`, and leaves out `source_id`. Change it on purpose, never
  as a side effect of changing `models.Envelope` or the bus's wire format.
- The signature covers the body bytes as sent. Re-marshalling the body after
  signing, or adding fields between signing and sending, breaks every
  consumer's verification.
- Errors never carry the webhook URL, the secret, or the signature, since a URL
  can hold a token in its query string. `http.Client.Do` wraps its errors in a
  `*url.Error` that names the URL, which is why `Deliver` unwraps it. Keep it
  that way when adding error paths, and when logging deliveries.

## Known gaps

- **Nothing delivers yet:** `Deliverer` and `Bus.Consume` exist, but no worker
  joins them, and `relay serve` doesn't start one. **Until:** the delivery
  worker is built.
- **A failing webhook is retried in a tight loop:** there's no backoff, so a
  down endpoint is called again as fast as it fails, and the destination's
  queue waits behind that one envelope. **Until:** not decided.
- **The hmac algorithm and encoding are only checked at delivery:** creating a
  destination accepts any string, but delivery supports only `sha256` and
  `sha512`, and `hex` and `base64`, so a destination with any other value fails
  every delivery. **Until:** not decided.
- **No guard on where webhooks point:** a destination's URL can be a private or
  loopback address, such as a cloud metadata endpoint, and Relay will POST to
  it. **Until:** not decided.
