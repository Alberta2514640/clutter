// lib/features/organization/keys.ts

export const orgKeys = {
  base: ["organization"] as const,

  organization: () => [...orgKeys.base, "details"] as const,
  members: () => [...orgKeys.base, "members"] as const,
  availableUsers: () => [...orgKeys.base, "availableUsers"] as const,
};
