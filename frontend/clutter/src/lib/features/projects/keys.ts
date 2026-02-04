export const projectKeys = {
  all: ["projects"] as const,
  list: (token: string) => [...projectKeys.all, "list", token] as const,
  byId: (projectId: string) => [...projectKeys.all, "byId", projectId] as const,
};
