export const projectKeys = {
  all: ["projects"] as const,
  list: (tenantId: string) => [...projectKeys.all, "list", tenantId] as const,
  byId: (projectId: string) => [...projectKeys.all, "byId", projectId] as const,
};
