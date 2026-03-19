// lib/features/organization/keys.ts
export const orgKeys = {
  base: ["organization"] as const,
  list: () => [...orgKeys.base, "list"] as const,
  cloudFormationStackUrl: () => [...orgKeys.base, "cloudformation-stack-url"] as const,
  accounts: (organizationId: string) => [...orgKeys.base, "accounts", organizationId] as const,
};
