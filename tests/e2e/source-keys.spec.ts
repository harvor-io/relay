import { expect, test } from "@playwright/test";
import { createSource, createSourceKey } from "./helpers";

test("POST /sources/{id}/keys issues a key and returns its secret", async ({ request }) => {
  const source = await createSource(request);

  const response = await request.post(`sources/${source.id}/keys`, { data: { name: "e2e key" } });

  expect(response.status()).toBe(201);
  const body = await response.json();
  expect(body).toMatchObject({ source_id: source.id, name: "e2e key", is_active: true });
  expect(body.secret).toBeTruthy();
});

test("GET /sources/{id}/keys lists a source's keys", async ({ request }) => {
  const source = await createSource(request);
  const key = await createSourceKey(request, source.id);

  const response = await request.get(`sources/${source.id}/keys`);

  expect(response.status()).toBe(200);
  const body: Array<{ id: string }> = await response.json();
  expect(body.some((item) => item.id === key.id)).toBe(true);
});

test("GET /sources/{id}/keys/{key_id} returns a key", async ({ request }) => {
  const source = await createSource(request);
  const key = await createSourceKey(request, source.id);

  const response = await request.get(`sources/${source.id}/keys/${key.id}`);

  expect(response.status()).toBe(200);
  expect(await response.json()).toMatchObject({ id: key.id, source_id: source.id, is_active: true });
});

test("POST /sources/{id}/keys/{key_id}/deactivate deactivates a key", async ({ request }) => {
  const source = await createSource(request);
  const key = await createSourceKey(request, source.id);

  const response = await request.post(`sources/${source.id}/keys/${key.id}/deactivate`);

  expect(response.status()).toBe(200);
  expect(await response.json()).toMatchObject({ id: key.id, is_active: false });
});

test("POST /sources/{id}/keys/{key_id}/activate activates a key", async ({ request }) => {
  const source = await createSource(request);
  const key = await createSourceKey(request, source.id);
  await request.post(`sources/${source.id}/keys/${key.id}/deactivate`);

  const response = await request.post(`sources/${source.id}/keys/${key.id}/activate`);

  expect(response.status()).toBe(200);
  expect(await response.json()).toMatchObject({ id: key.id, is_active: true });
});

test("DELETE /sources/{id}/keys/{key_id} deletes a key", async ({ request }) => {
  const source = await createSource(request);
  const key = await createSourceKey(request, source.id);

  const response = await request.delete(`sources/${source.id}/keys/${key.id}`);

  expect(response.status()).toBe(204);
});
