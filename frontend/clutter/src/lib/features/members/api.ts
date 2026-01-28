// lib/features/members/api.ts
import type { AvailableUser, OrgMember } from "./types";

// --- Mock data (members-only for now) ---
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
const clone = <T>(v: T): T => JSON.parse(JSON.stringify(v)) as T;

export const membersApi = {
  list: async (): Promise<OrgMember[]> => {
    await sleep(250);
    return clone(MOCK_MEMBERS);
  },

  add: async (userId: string, role: string): Promise<OrgMember> => {
    await sleep(450);

    const user = MOCK_AVAILABLE_USERS.find((u) => u.id === userId);
    if (!user) throw new Error("User not found");

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
