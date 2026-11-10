import { expect, test } from "@playwright/test";
import { createSource, createSourceKey, signSourceEventBody } from "./helpers";

// Which events a source accepts is a business rule, checked by the acceptance
// tests in tests/acceptance/features/events.feature, not here.

test("POST /sources/{slug}/events accepts a signed event", async ({ request }) => {
  const source = await createSource(request);
  const key = await createSourceKey(request, source.id);
  const data = { hello: "world" };
  const body = JSON.stringify({ type: "e2e.event", data });

  const response = await request.post(`sources/${source.slug}/events`, {
    data: body,
    headers: { "X-Relay-Signature": signSourceEventBody(key.secret, body) },
  });

  expect(response.status()).toBe(201);
  const envelope = await response.json();
  expect(envelope).toMatchObject({
    source_id: source.id,
    source: source.slug,
    type: "e2e.event",
    topic: `${source.slug}.e2e.event`,
    data,
  });
  expect(envelope.id).toBeTruthy();
  expect(envelope.created_at).toBeTruthy();
});

test("POST /sources/{id}/events accepts a signed event", async ({ request }) => {
  const source = await createSource(request);
  const key = await createSourceKey(request, source.id);
  const data = { hello: "again" };
  const body = JSON.stringify({ type: "e2e.event", data });

  const response = await request.post(`sources/${source.id}/events`, {
    data: body,
    headers: { "X-Relay-Signature": signSourceEventBody(key.secret, body) },
  });

  expect(response.status()).toBe(201);
  expect(await response.json()).toMatchObject({ source_id: source.id, topic: `${source.slug}.e2e.event`, data });
});
