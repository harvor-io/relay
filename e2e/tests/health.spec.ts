import { expect, test } from "@playwright/test";

test("GET /healthz reports the service is up", async ({ request }) => {
  const response = await request.get("healthz");

  expect(response.status()).toBe(200);
  expect(await response.json()).toEqual({ status: "OK" });
});
