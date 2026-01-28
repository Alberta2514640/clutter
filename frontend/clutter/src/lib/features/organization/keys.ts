// lib/features/organization/keys.ts
export const orgKeys = {
  base: ["organization"] as const,
  list: () => [...orgKeys.base, "list"] as const,
};
