import { defineConfig } from "@playwright/test";

// These are black-box tests: they expect a Relay server already running
// somewhere (`make run`, `make up`, ...) and never start or manage one
// themselves. Point BASE_URL elsewhere to test a different instance.
// Trailing slash matters: APIRequestContext resolves relative request paths
// against baseURL with WHATWG URL rules, where a leading "/" on the request
// path resets to the origin root and drops the "/api/v1" prefix entirely.
// Keep the trailing slash here and never start a request path with "/".
const BASE_URL = process.env.BASE_URL ?? "http://localhost:8080/api/v1/";

export default defineConfig({
  testDir: "./tests",
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: "list",
  use: {
    baseURL: BASE_URL,
    extraHTTPHeaders: { "Content-Type": "application/json" },
  },
});
