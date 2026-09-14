import { expect, test } from "@playwright/test";
import { createSource, createSourceKey, signIngestBody } from "./helpers";
import type { CreatedSourceKeyResource, SourceResource } from "./helpers";

test.describe("ingest", () => {
  let source: SourceResource;
  let key: CreatedSourceKeyResource;

  test.beforeAll(async ({ request }) => {
    source = await createSource(request);
    key = await createSourceKey(request, source.id);
  });

  test("401 when the signature does not match any active key", async ({ request }) => {
    const body = JSON.stringify({ type: "e2e.event", data: { hello: "world" } });

    const response = await request.post(`ingest/${source.slug}`, {
      data: body,
      headers: { "X-Relay-Signature": "0".repeat(64) },
    });

    expect(response.status()).toBe(401);
    expect(await response.json()).toMatchObject({ error: expect.any(String) });
  });

  test("happy path: ingest by source slug", async ({ request }) => {
    const payload = { hello: "world" };
    const body = JSON.stringify({ type: "e2e.event", data: payload });
    const signature = signIngestBody(key.secret, body);

    const response = await request.post(`ingest/${source.slug}`, {
      data: body,
      headers: { "X-Relay-Signature": signature },
    });

    expect(response.status()).toBe(201);
    const envelope = await response.json();
    expect(envelope).toMatchObject({
      source_id: source.id,
      source: source.slug,
      type: "e2e.event",
      topic: `${source.slug}.e2e.event`,
      data: payload,
    });
    expect(envelope.id).toBeTruthy();
    expect(envelope.created_at).toBeTruthy();
  });

  test("happy path: ingest by source id", async ({ request }) => {
    const payload = { hello: "again" };
    const body = JSON.stringify({ type: "e2e.event", data: payload });
    const signature = signIngestBody(key.secret, body);

    const response = await request.post(`ingest/${source.id}`, {
      data: body,
      headers: { "X-Relay-Signature": signature },
    });

    expect(response.status()).toBe(201);
    const envelope = await response.json();
    expect(envelope).toMatchObject({
      source_id: source.id,
      source: source.slug,
      type: "e2e.event",
      topic: `${source.slug}.e2e.event`,
      data: payload,
    });
    expect(envelope.id).toBeTruthy();
    expect(envelope.created_at).toBeTruthy();
  });
});
