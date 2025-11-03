export const qk = {
  tenant: (tenantId: string) => ["tenant", tenantId] as const,
  projects: (tenantId: string) => ["projects", tenantId] as const,
  project: (projectId: string) => ["project", projectId] as const,
  canvasBundle: (projectId: string, canvasId: string) => ["canvasBundle", projectId, canvasId] as const,
  workspaces: (tenantId: string) => ["workspaces", tenantId] as const,
  workspace: (projectId: string, workspaceId: string) => ["workspace", projectId, workspaceId] as const,
  moduleSet: (projectId: string, moduleSetId: string) => ["moduleSet", projectId, moduleSetId] as const,
  varSets: (projectId: string, env?: string) => ["varSets", projectId, env] as const,
  runs: (projectId: string) => ["runs", projectId] as const,
  run: (projectId: string, runId: string) => ["run", projectId, runId] as const,
};
