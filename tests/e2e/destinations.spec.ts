import { expect, test } from "@playwright/test";
import { createDestination, uniqueSlug } from "./helpers";

test("POST /destinations registers a signed webhook destination", async ({ request }) => {
  const name = `E2E Destination ${uniqueSlug("e2e")}`;
  const description = "created by the Playwright e2e suite";
  const hmacFields = {
    algorithm: "sha256",
    signature_header: "X-Signature",
    encoding: "hex",
    signing_template: "{timestamp}.{body}",
    timestamp_header: "X-Timestamp",
  };
  // The request supplies the plaintext signing secret; the response carries a
  // secret_id reference in its place, never the plaintext.
  const config = {
    url: "https://example.com/webhook",
    auth: { type: "hmac", config: { secret: "e2e-signing-secret", ...hmacFields } },
  };

  const response = await request.post("destinations", { data: { name, description, config } });

  expect(response.status()).toBe(201);
  const body = await response.json();
  expect(body).toMatchObject({
    name,
    description,
    type: "webhook",
    config: { url: config.url, auth: { type: "hmac", config: hmacFields } },
    is_active: true,
  });
  expect(body.config.auth.config.secret_id).toBeTruthy();
  expect(body.config.auth.config.secret).toBeUndefined();
});

test("GET /destinations lists destinations", async ({ request }) => {
  const destination = await createDestination(request);

  const response = await request.get("destinations");

  expect(response.status()).toBe(200);
  const body: Array<{ id: string }> = await response.json();
  expect(body.some((item) => item.id === destination.id)).toBe(true);
});

test("GET /destinations/{id} returns a destination", async ({ request }) => {
  const destination = await createDestination(request);

  const response = await request.get(`destinations/${destination.id}`);

  expect(response.status()).toBe(200);
  expect(await response.json()).toMatchObject({
    id: destination.id,
    name: destination.name,
    config: { url: destination.config.url },
  });
});

test("PATCH /destinations/{id} renames a destination and changes its description", async ({ request }) => {
  const destination = await createDestination(request);
  const name = `${destination.name} (updated)`;
  const description = "updated by the Playwright e2e suite";

  const response = await request.patch(`destinations/${destination.id}`, { data: { name, description } });

  expect(response.status()).toBe(200);
  expect(await response.json()).toMatchObject({ id: destination.id, name, description });
});

test("POST /destinations/{id}/deactivate deactivates a destination", async ({ request }) => {
  const destination = await createDestination(request);

  const response = await request.post(`destinations/${destination.id}/deactivate`);

  expect(response.status()).toBe(200);
  expect(await response.json()).toMatchObject({ id: destination.id, is_active: false });
});

test("POST /destinations/{id}/activate activates a destination", async ({ request }) => {
  const destination = await createDestination(request);
  await request.post(`destinations/${destination.id}/deactivate`);

  const response = await request.post(`destinations/${destination.id}/activate`);

  expect(response.status()).toBe(200);
  expect(await response.json()).toMatchObject({ id: destination.id, is_active: true });
});

test("DELETE /destinations/{id} deletes a destination", async ({ request }) => {
  const destination = await createDestination(request);

  const response = await request.delete(`destinations/${destination.id}`);

  expect(response.status()).toBe(204);
});
