export const deploymentKeys = {
  base: ["deployments"] as const,
  forDiagram: (organizationId: string, diagramId: string) => [...deploymentKeys.base, "diagram", organizationId, diagramId] as const,
  compatibility: (organizationId: string, deploymentId: string) =>
    [...deploymentKeys.base, "compatibility", organizationId, deploymentId] as const,
  preview: (organizationId: string, deploymentId: string) => [...deploymentKeys.base, "preview", organizationId, deploymentId] as const,
  runLogs: (organizationId: string, runId: string) => [...deploymentKeys.base, "run-logs", organizationId, runId] as const,
};
