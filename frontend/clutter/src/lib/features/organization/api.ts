// lib/features/organization/api.ts
// lib/features/organization/api.ts
import type { AvailableUser, Organization, OrgMember } from "./types";

// --- Mock data ---
let MOCK_ORGANIZATION: Organization | null = {
  tenantId: "t_demo_001",
  name: "Demo Organization",
  slug: "demo-org",
  timeZone: "America/Edmonton",
  createdAt: "2025-01-01T00:00:00Z",
  updatedAt: "2025-01-20T10:00:00Z",
};

let MOCK_MEMBERS: OrgMember[] = [
  {
    id: "m1",
    userId: "demo_user_001",
    name: "Hamza Amar",
    email: "hamza.amar@ucalgary.ca",
    role: "Project Admin",
  },
];

const MOCK_AVAILABLE_USERS: AvailableUser[] = [
  { id: "2", name: "Santiago Fuentes", email: "santiago@ucalgary.ca" },
  { id: "3", name: "Alice Johnson", email: "alice@ucalgary.ca" },
  { id: "4", name: "Bob Smith", email: "bob@ucalgary.ca" },
];

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
const nowIso = () => new Date().toISOString();
const clone = <T>(v: T): T => JSON.parse(JSON.stringify(v)) as T;

const makeTenantId = () => `t_${crypto.randomUUID()}`;

// If you don’t want crypto.randomUUID() for older envs:
// const makeTenantId = () => `t_${Date.now()}_${Math.random().toString(16).slice(2)}`;

export type CreateOrganizationInput = {
  name: string;
  slug: string;
  timeZone?: string;
  // optional: who becomes the initial admin
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

    // Optional rule: block creating if one already exists
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

    // Optional: create initial member (owner/admin)
    if (input.ownerUser) {
      MOCK_MEMBERS = [
        {
          id: `m_${crypto.randomUUID()}`,
          userId: input.ownerUser.userId,
          name: input.ownerUser.name,
          email: input.ownerUser.email,
          role: "Project Admin",
        },
      ];
    } else {
      // Or keep existing / empty depending on your flow
      MOCK_MEMBERS = [];
    }

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
    MOCK_MEMBERS = [];
  },
};

export const membersApi = {
  list: async (): Promise<OrgMember[]> => {
    await sleep(250);
    return clone(MOCK_MEMBERS);
  },

  add: async (userId: string, role: string): Promise<OrgMember> => {
    await sleep(450);

    const user = MOCK_AVAILABLE_USERS.find((u) => u.id === userId);
    if (!user) throw new Error("User not found");

    // Prevent duplicates (optional but nice)
    const already = MOCK_MEMBERS.some((m) => m.userId === userId);
    if (already) throw new Error("User is already a member");

    const newMember: OrgMember = {
      id: `m_${Date.now()}`,
      userId,
      name: user.name,
      email: user.email,
      role,
    };

    MOCK_MEMBERS = [...MOCK_MEMBERS, newMember];
    return clone(newMember);
  },

  remove: async (memberId: string): Promise<void> => {
    await sleep(450);
    MOCK_MEMBERS = MOCK_MEMBERS.filter((m) => m.id !== memberId);
  },

  updateRole: async (memberId: string, role: string): Promise<OrgMember> => {
    await sleep(450);

    const idx = MOCK_MEMBERS.findIndex((m) => m.id === memberId);
    if (idx === -1) throw new Error("Member not found");

    const updated: OrgMember = { ...MOCK_MEMBERS[idx], role };
    MOCK_MEMBERS = MOCK_MEMBERS.map((m) => (m.id === memberId ? updated : m));
    return clone(updated);
  },
};

export const availableUsersApi = {
  list: async (): Promise<AvailableUser[]> => {
    await sleep(250);
    return clone(MOCK_AVAILABLE_USERS);
  },
};
