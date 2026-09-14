import type { APIRequestContext } from "@playwright/test";
import { randomUUID } from "node:crypto";

export interface SourceResource {
  id: string;
  name: string;
  slug: string;
  description: string | null;
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
