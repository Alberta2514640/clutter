export const projectKeys = {
  base: ["projects"] as const,
  list: (tenantId: string) => [...projectKeys.base, "list", tenantId] as const,
};
