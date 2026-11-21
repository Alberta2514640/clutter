export const functionCodeKeys = {
  base: ["functionCode"] as const,
  forDiagram: (organizationId: string, diagramId: string) => [...functionCodeKeys.base, "diagram", organizationId, diagramId] as const,
};
