import { expect, test } from "@playwright/test";
import { uniqueSlug } from "./helpers";

test("destinations CRUD and activation lifecycle happy path", async ({ request }) => {
  const slug = uniqueSlug("e2e-destination");
  const name = `E2E Destination ${slug}`;
  const description = "created by the Playwright e2e suite";
  const hmacFields = {
    algorithm: "sha256",
    signature_header: "X-Signature",
    encoding: "hex",
    signing_template: "{timestamp}.{body}",
    timestamp_header: "X-Timestamp",
  };
  // The request supplies the plaintext signing secret; the server encrypts
  // it, stores it, and returns a secret_id reference in its place, never the
  // plaintext.
  const config = {
    url: "https://example.com/webhook",
    auth: { type: "hmac", config: { secret: "e2e-signing-secret", ...hmacFields } },
  };
  let id: string;

  await test.step("create", async () => {
    const response = await request.post("destinations", {
      data: { name, description, config },
    });

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
    id = body.id;
  });

  await test.step("get by id", async () => {
    const response = await request.get(`destinations/${id}`);

    expect(response.status()).toBe(200);
    expect(await response.json()).toMatchObject({
      id,
      name,
      description,
      config: { url: config.url, auth: { type: "hmac", config: hmacFields } },
    });
  });

  await test.step("list includes the created destination", async () => {
    const response = await request.get("destinations");

    expect(response.status()).toBe(200);
    const body: Array<{ id: string }> = await response.json();
    expect(body.some((destination) => destination.id === id)).toBe(true);
  });

  await test.step("update name and description", async () => {
    const updatedName = `${name} (updated)`;
    const updatedDescription = "updated by the Playwright e2e suite";
    const response = await request.patch(`destinations/${id}`, {
      data: { name: updatedName, description: updatedDescription },
    });

    expect(response.status()).toBe(200);
    expect(await response.json()).toMatchObject({
      id,
      name: updatedName,
      description: updatedDescription,
    });
  });

  await test.step("deactivate", async () => {
    const response = await request.post(`destinations/${id}/deactivate`);

    expect(response.status()).toBe(200);
    expect(await response.json()).toMatchObject({ id, is_active: false });
  });

  await test.step("activate", async () => {
    const response = await request.post(`destinations/${id}/activate`);

    expect(response.status()).toBe(200);
    expect(await response.json()).toMatchObject({ id, is_active: true });
  });

  await test.step("delete", async () => {
    const response = await request.delete(`destinations/${id}`);

    expect(response.status()).toBe(204);
  });
});
