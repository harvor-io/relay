import type { APIRequestContext } from "@playwright/test";
import { createHmac, randomUUID } from "node:crypto";

export interface SourceResource {
  id: string;
  name: string;
  slug: string;
  description: string | null;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface SourceKeyResource {
  id: string;
  source_id: string;
  name: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreatedSourceKeyResource extends SourceKeyResource {
  secret: string;
}

export interface HMACAuthConfig {
  secret_id: string;
  algorithm: string;
  signature_header: string;
  encoding: string;
  signing_template: string;
  timestamp_header: string;
}

export interface AuthConfig {
  type: "hmac";
  config: HMACAuthConfig;
}

export interface WebhookConfig {
  url: string;
  auth?: AuthConfig;
}

export interface DestinationResource {
  id: string;
  name: string;
  type: "webhook";
  config: WebhookConfig;
  description: string | null;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

// uniqueSlug avoids collisions between test runs sharing the same database.
export function uniqueSlug(prefix: string): string {
  return `${prefix}-${randomUUID()}`;
}

export async function createSource(
  request: APIRequestContext,
  overrides: Partial<{ name: string; slug: string; description: string | null }> = {},
): Promise<SourceResource> {
  const slug = overrides.slug ?? uniqueSlug("e2e-source");
  const response = await request.post("sources", {
    data: {
      name: overrides.name ?? `E2E Source ${slug}`,
      slug,
      description: overrides.description ?? null,
    },
  });
  if (response.status() !== 201) {
    throw new Error(`createSource: expected 201, got ${response.status()}: ${await response.text()}`);
  }
  return response.json();
}

export async function createSourceKey(
  request: APIRequestContext,
  sourceID: string,
  overrides: Partial<{ name: string }> = {},
): Promise<CreatedSourceKeyResource> {
  const response = await request.post(`sources/${sourceID}/keys`, {
    data: { name: overrides.name ?? "e2e key" },
  });
  if (response.status() !== 201) {
    throw new Error(`createSourceKey: expected 201, got ${response.status()}: ${await response.text()}`);
  }
  return response.json();
}

// signIngestBody computes the X-Relay-Signature header value for body: a
// hex-encoded HMAC-SHA256 keyed with a source key's plaintext secret (itself
// hex-encoded text, used as-is, not decoded to raw bytes).
export function signIngestBody(secret: string, body: string): string {
  return createHmac("sha256", secret).update(body).digest("hex");
}
