import { defineConfig } from "@playwright/test";
import { tmpdir } from "node:os";
import { join } from "node:path";

/*
 * End-to-end tests call Relay over HTTP, black box, one endpoint's happy path
 * each, against the app and the local stack's RabbitMQ (RABBITMQ_URI, the
 * app's default otherwise). They need no browser. Playwright builds and starts
 * the app and waits for it to respond, or reuses one already running locally
 * (`make run`, `make up`). Point BASE_URL elsewhere to test another instance.
 *
 * Trailing slash matters: request paths resolve against baseURL with WHATWG
 * URL rules, where a leading "/" resets to the origin and drops "/api/v1".
 * Keep the trailing slash here and never start a request path with "/".
 */
const BASE_URL = process.env.BASE_URL ?? "http://localhost:8080/api/v1/";

export default defineConfig({
  testDir: "tests/e2e",
  forbidOnly: !!process.env.CI,
  reporter: process.env.CI ? "github" : "list",
  use: {
    baseURL: BASE_URL,
    extraHTTPHeaders: { "Content-Type": "application/json" },
  },
  webServer: {
    command: "go build -o bin/relay ./cmd/relay && bin/relay serve",
    url: new URL("readyz", BASE_URL).toString(),
    reuseExistingServer: !process.env.CI,
    timeout: 180_000,
    env: {
      PORT: "8080",
      LOG_FORMAT: "text",
      DATABASE_URL: `file:${join(tmpdir(), "relay-e2e.db")}`,
      // Test-only key; nothing encrypted with it outlives the test database.
      SECRETS_ENCRYPTION_KEY: "PMR/hf5+zhUN5AMEnBHC3l+2n7jcms8f2EfpuFf9MvU=",
    },
  },
});
