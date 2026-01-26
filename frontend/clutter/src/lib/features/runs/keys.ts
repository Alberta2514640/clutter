export const runKeys = {
  base: ["runs"] as const,
  recent: (tenantId: string) => [...runKeys.base, "recent", tenantId] as const,
};
