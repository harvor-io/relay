import { expect, test } from "@playwright/test";

test("GET /docs/api serves the API documentation", async ({ request, baseURL }) => {
  // The docs live at the origin's root, outside the /api/v1 base URL.
  const response = await request.get(new URL("/docs/api", baseURL).toString());

  expect(response.status()).toBe(200);
  expect(response.headers()["content-type"]).toContain("text/html");
});
