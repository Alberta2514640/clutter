// lib/features/members/api.ts
import type { UserData } from "@/lib/features/user/types"; // adjust path
import type { AvailableUser, OrgMember } from "./types";

let MOCK_MEMBERS: OrgMember[] = [];

const MOCK_AVAILABLE_USERS: AvailableUser[] = [
  { id: "3", name: "Alice Johnson", email: "alice@ucalgary.ca" },
  { id: "4", name: "Bob Smith", email: "bob@ucalgary.ca" },
];

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
const clone = <T>(v: T): T => JSON.parse(JSON.stringify(v)) as T;

function ensureAvailableUser(user: AvailableUser) {
  const exists = MOCK_AVAILABLE_USERS.some((u) => u.id === user.id);
  if (!exists) MOCK_AVAILABLE_USERS.unshift(user);
}

export const membersApi = {
  bootstrapWithMe: async (me: UserData | null | undefined): Promise<void> => {
    if (!me?.userId) return;

    // Ensure "me" is selectable in the "available users" list too
    ensureAvailableUser({
      id: me.userId,
      name: me.displayName || me.email,
      email: me.email,
    });

    // Ensure "me" exists as a member
    const already = MOCK_MEMBERS.some((m) => m.userId === me.userId);
    if (already) return;

    MOCK_MEMBERS = [
      {
        id: `m_${me.userId}`, // stable, prevents dupes
        userId: me.userId,
        name: me.displayName || me.email,
        email: me.email,
        role: "Project Admin",
      },
      ...MOCK_MEMBERS,
    ];
  },

  reset: async (): Promise<void> => {
    MOCK_MEMBERS = [];
  },

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
