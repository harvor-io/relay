import { expect, test } from "@playwright/test";

test("GET /livez reports the server is up", async ({ request }) => {
  const response = await request.get("livez");

  expect(response.status()).toBe(200);
  expect(await response.json()).toMatchObject({
    status: "up",
    details: { http: { status: "up" } },
  });
});

test("GET /readyz reports the server and its dependencies are up", async ({
  request,
}) => {
  const response = await request.get("readyz");

  expect(response.status()).toBe(200);
  expect(await response.json()).toMatchObject({
    status: "up",
    details: {
      http: { status: "up" },
      database: { status: "up" },
      secrets: { status: "up" },
      eventbus: { status: "up" },
    },
  });
});

test("GET /healthz reports the service is up", async ({ request }) => {
  const response = await request.get("healthz");

  expect(response.status()).toBe(200);
  expect(await response.json()).toMatchObject({ status: "up" });
});
