// lib/features/logs/keys.ts

export const logKeys = {
  base: ["logs"] as const,

  // list of log files for a diagram
  files: (orgId: string, projId: string, diagramId: string) =>
    [...logKeys.base, "files", orgId, projId, diagramId] as const,

  // presigned URL for a single file
  fileUrl: (orgId: string, projId: string, diagramId: string, deploymentId: string, file: string) =>
    [...logKeys.base, "fileUrl", orgId, projId, diagramId, deploymentId, file] as const,

  // live log stream for a running task
  live: (orgId: string, projId: string, diagramId: string, taskArn: string) =>
    [...logKeys.base, "live", orgId, projId, diagramId, taskArn] as const,

  recentActivity: (orgId: string, diagramId: string) => ["logs", "recent-activity", orgId, diagramId] as const,
};
