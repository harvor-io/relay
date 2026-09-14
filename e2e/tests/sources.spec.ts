import { expect, test } from "@playwright/test";
import { uniqueSlug } from "./helpers";

test("sources CRUD happy path", async ({ request }) => {
  const slug = uniqueSlug("e2e-source");
  const name = `E2E Source ${slug}`;
  const description = "created by the Playwright e2e suite";
  let id: string;

  await test.step("create", async () => {
    const response = await request.post("sources", {
      data: { name, slug, description },
    });

    expect(response.status()).toBe(201);
    const body = await response.json();
    expect(body).toMatchObject({ name, slug, description });
    expect(body.id).toBeTruthy();
    id = body.id;
  });

  await test.step("get by id", async () => {
    const response = await request.get(`sources/${id}`);

    expect(response.status()).toBe(200);
    expect(await response.json()).toMatchObject({ id, name, slug });
  });

  await test.step("list includes the created source", async () => {
    const response = await request.get("sources");

    expect(response.status()).toBe(200);
    const body: Array<{ id: string }> = await response.json();
    expect(body.some((source) => source.id === id)).toBe(true);
  });

  await test.step("delete", async () => {
    const response = await request.delete(`sources/${id}`);

    expect(response.status()).toBe(204);
  });
});
