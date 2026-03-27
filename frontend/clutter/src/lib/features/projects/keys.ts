// lib/features/projects/keys.ts
export const projectKeys = {
  all: ["projects"] as const,
  // lists (scoped by organizationId)
  lists: () => [...projectKeys.all, "list"] as const,
  list: (organizationId: string) => [...projectKeys.lists(), organizationId] as const,
  // single project
  details: () => [...projectKeys.all, "detail"] as const,
  detail: (organizationId: string, projectId: string) => [...projectKeys.details(), organizationId, projectId] as const,
  terraformLogs: (orgId: string, projId: string, diagramId: string) =>
    [...projectKeys.all, "terraform-logs", orgId, projId, diagramId] as const,
};
