export const catalogKeys = {
  base: ["catalog"] as const,
  list: () => [...catalogKeys.base, "list"] as const,
  linked: (organizationId: string, connector: string) => [...catalogKeys.base, "linked", organizationId, connector] as const,
};
