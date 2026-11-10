# Secrets

How Relay keeps the secrets it has to be able to read back: source key secrets,
and the signing secrets of webhook destinations.

**Code:** `pkg/secrets` (the `Store` and its SQLite implementation),
`pkg/encryptor/aes`, `internal/services/source_keys.go`, and
`resolveHMACAuthConfig` in `internal/services/destinations.go`. Wired up in
`internal/commands/serve.go`.

## How it works

Secret values are encrypted with AES using `SECRETS_ENCRYPTION_KEY`, then kept
in a secret store that can live in its own database (`SECRETS_DATABASE_*`) or
share the main one. Records elsewhere hold only the secret's ID: `source_keys`
has a `secret_id` column, and a webhook destination's config stores
`auth.config.secret_id` in place of the secret the caller sent.

- **Source keys:** Relay generates 32 random bytes, stores them encrypted, and
  returns them hex-encoded once, in the create response. To verify an event,
  `VerifySignature` decrypts every active key of the source and checks the
  `X-Relay-Signature` header against an HMAC-SHA256 of the request body.
- **Destination signing secrets:** the caller supplies the secret when creating
  the destination. Nothing reads it back yet, since delivery isn't built.

## Gotchas

- The HMAC key for event signatures is the hex string handed to the producer,
  used as bytes, not the 32 bytes it decodes to. Producers sign with the string
  exactly as they got it, so don't "fix" this to decode first; every existing
  producer would start failing.
- Losing or changing `SECRETS_ENCRYPTION_KEY` makes every stored secret
  unreadable. Every source key would have to be reissued and every signed
  destination recreated.
- The secret store opens its database with `DATABASE_AUTH_TOKEN`, the main
  database's token. A hosted secrets database has to accept the same token.
- Deleting a key or destination removes its row first and its secret second, on
  a best-effort basis. A failure in between leaves an orphaned secret, which is
  harmless but never cleaned up.
- Verifying a signature decrypts every active key of the source on every event.
  A source with many active keys pays for each one.

## Known gaps

- **No rotation of the encryption key:** there's no way to re-encrypt stored
  secrets under a new `SECRETS_ENCRYPTION_KEY`. **Until:** not decided.
