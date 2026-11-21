export const deploymentKeys = {
  base: ["deployments"] as const,
  forDiagram: (organizationId: string, diagramId: string) => [...deploymentKeys.base, "diagram", organizationId, diagramId] as const,
  // Every list of an organization's deployments (all projects, or one).
  organizations: () => [...deploymentKeys.base, "organization"] as const,
  forOrganization: (organizationId: string, projectId: string) => [...deploymentKeys.organizations(), organizationId, projectId] as const,
  compatibility: (organizationId: string, deploymentId: string) =>
    [...deploymentKeys.base, "compatibility", organizationId, deploymentId] as const,
  preview: (organizationId: string, deploymentId: string) => [...deploymentKeys.base, "preview", organizationId, deploymentId] as const,
  runLogs: (organizationId: string, runId: string) => [...deploymentKeys.base, "run-logs", organizationId, runId] as const,
};
