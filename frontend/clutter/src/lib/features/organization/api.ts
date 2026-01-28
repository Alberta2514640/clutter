// lib/features/organization/api.ts
import type { Organization } from "./types";

let MOCK_ORGANIZATION: Organization | null = {
  tenantId: "t_demo_001",
  name: "Demo Organization",
  slug: "demo-org",
  timeZone: "America/Edmonton",
  createdAt: "2025-01-01T00:00:00Z",
  updatedAt: "2025-01-20T10:00:00Z",
};

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
const nowIso = () => new Date().toISOString();
const clone = <T>(v: T): T => JSON.parse(JSON.stringify(v)) as T;

const makeTenantId = () => `t_${crypto.randomUUID()}`;

export type CreateOrganizationInput = {
  name: string;
  slug: string;
  timeZone?: string;
  ownerUser?: { userId: string; name: string; email: string };
};

export const organizationApi = {
  get: async (): Promise<Organization | null> => {
    await sleep(250);
    return clone(MOCK_ORGANIZATION);
  },

  create: async (input: CreateOrganizationInput): Promise<Organization> => {
    await sleep(450);

    const name = input.name.trim();
    const slug = input.slug.trim();
    const timeZone = input.timeZone?.trim() || "America/Edmonton";

    if (!name) throw new Error("Organization name is required");
    if (!slug) throw new Error("Organization slug is required");

    if (MOCK_ORGANIZATION && MOCK_ORGANIZATION.tenantId !== "t_deleted") {
      throw new Error("Organization already exists");
    }

    const createdAt = nowIso();
    const tenantId = makeTenantId();

    MOCK_ORGANIZATION = {
      tenantId,
      name,
      slug,
      timeZone,
      createdAt,
      updatedAt: createdAt,
    };

    return clone(MOCK_ORGANIZATION);
  },

  update: async (data: Partial<Organization>): Promise<Organization> => {
    await sleep(450);
    if (!MOCK_ORGANIZATION) throw new Error("No organization to update");

    MOCK_ORGANIZATION = {
      ...MOCK_ORGANIZATION,
      ...data,
      updatedAt: nowIso(),
    };
    return clone(MOCK_ORGANIZATION);
  },

  delete: async (): Promise<void> => {
    await sleep(450);
    MOCK_ORGANIZATION = null;
  },
};
