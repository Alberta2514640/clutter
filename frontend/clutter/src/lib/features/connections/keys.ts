export const connectionKeys = {
  base: ["connections"] as const,
  list: (organizationId: string) => [...connectionKeys.base, "list", organizationId] as const,
  options: (organizationId: string, connectionId: string) => [...connectionKeys.base, "options", organizationId, connectionId] as const,
};
