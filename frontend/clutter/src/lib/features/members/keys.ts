// lib/features/members/keys.ts
export const membersKeys = {
  base: ["members"] as const,
  list: () => [...membersKeys.base, "list"] as const,
  availableUsers: () => [...membersKeys.base, "availableUsers"] as const,
};
