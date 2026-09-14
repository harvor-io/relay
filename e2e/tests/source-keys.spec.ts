import { expect, test } from "@playwright/test";
import { createSource } from "./helpers";
import type { SourceResource } from "./helpers";

test.describe("source keys happy path", () => {
  let source: SourceResource;

  test.beforeAll(async ({ request }) => {
    source = await createSource(request);
  });

  test("keys CRUD and activation lifecycle", async ({ request }) => {
    let id: string;

    await test.step("create", async () => {
      const response = await request.post(`sources/${source.id}/keys`, {
        data: { name: "e2e key" },
      });

      expect(response.status()).toBe(201);
      const body = await response.json();
      expect(body).toMatchObject({ source_id: source.id, name: "e2e key", is_active: true });
      expect(body.secret).toBeTruthy();
      id = body.id;
    });

    await test.step("get by id", async () => {
      const response = await request.get(`sources/${source.id}/keys/${id}`);

      expect(response.status()).toBe(200);
      expect(await response.json()).toMatchObject({ id, is_active: true });
    });

    await test.step("list includes the created key", async () => {
      const response = await request.get(`sources/${source.id}/keys`);

      expect(response.status()).toBe(200);
      const body: Array<{ id: string }> = await response.json();
      expect(body.some((key) => key.id === id)).toBe(true);
    });

    await test.step("deactivate", async () => {
      const response = await request.post(`sources/${source.id}/keys/${id}/deactivate`);

      expect(response.status()).toBe(200);
      expect(await response.json()).toMatchObject({ id, is_active: false });
    });

    await test.step("activate", async () => {
      const response = await request.post(`sources/${source.id}/keys/${id}/activate`);

      expect(response.status()).toBe(200);
      expect(await response.json()).toMatchObject({ id, is_active: true });
    });

    await test.step("delete", async () => {
      const response = await request.delete(`sources/${source.id}/keys/${id}`);

      expect(response.status()).toBe(204);
    });
  });
});
