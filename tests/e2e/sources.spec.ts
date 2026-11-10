import { expect, test } from "@playwright/test";
import { createSource, uniqueSlug } from "./helpers";

test("POST /sources registers a source", async ({ request }) => {
  const slug = uniqueSlug("e2e-source");
  const name = `E2E Source ${slug}`;
  const description = "created by the Playwright e2e suite";

  const response = await request.post("sources", { data: { name, slug, description } });

  expect(response.status()).toBe(201);
  const body = await response.json();
  expect(body).toMatchObject({ name, slug, description, is_active: true });
  expect(body.id).toBeTruthy();
});

test("GET /sources lists sources", async ({ request }) => {
  const source = await createSource(request);

  const response = await request.get("sources");

  expect(response.status()).toBe(200);
  const body: Array<{ id: string }> = await response.json();
  expect(body.some((item) => item.id === source.id)).toBe(true);
});

test("GET /sources/{id} returns a source", async ({ request }) => {
  const source = await createSource(request);

  const response = await request.get(`sources/${source.id}`);

  expect(response.status()).toBe(200);
  expect(await response.json()).toMatchObject({ id: source.id, name: source.name, slug: source.slug });
});

test("PATCH /sources/{id} renames a source and changes its description", async ({ request }) => {
  const source = await createSource(request);
  const name = `${source.name} (updated)`;
  const description = "updated by the Playwright e2e suite";

  const response = await request.patch(`sources/${source.id}`, { data: { name, description } });

  expect(response.status()).toBe(200);
  expect(await response.json()).toMatchObject({ id: source.id, name, description, slug: source.slug });
});

test("POST /sources/{id}/deactivate deactivates a source", async ({ request }) => {
  const source = await createSource(request);

  const response = await request.post(`sources/${source.id}/deactivate`);

  expect(response.status()).toBe(200);
  expect(await response.json()).toMatchObject({ id: source.id, is_active: false });
});

test("POST /sources/{id}/activate activates a source", async ({ request }) => {
  const source = await createSource(request);
  await request.post(`sources/${source.id}/deactivate`);

  const response = await request.post(`sources/${source.id}/activate`);

  expect(response.status()).toBe(200);
  expect(await response.json()).toMatchObject({ id: source.id, is_active: true });
});

test("DELETE /sources/{id} deletes a source", async ({ request }) => {
  const source = await createSource(request);

  const response = await request.delete(`sources/${source.id}`);

  expect(response.status()).toBe(204);
});
