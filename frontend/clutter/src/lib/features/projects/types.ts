export interface DiagramSummary {
  id: string;
  name: string;
}
export interface Project {
  id: string;
  organizationId: string;
  creatorId: string;
  name: string;
  diagrams: DiagramSummary[];
  description: string;
  createdAt: string;
  createdBy: string;
}

export type CreateProjectInput = {
  organizationId: string;
  name: string;
  description?: string;
};

export type UpdateProjectInput = {
  organizationId?: string;
  name?: string;
  description?: string;
};

export type GetTerraformLogsInput = {
  orgId: string;
  projId: string;
  diagramId: string;
};

export type TerraformLogNode = {
  name?: string;
  type?: string;
  path?: string;
  children?: TerraformLogNode[];
  [key: string]: unknown;
};

export type TerraformLogsResponse =
  | TerraformLogNode
  | TerraformLogNode[]
  | Record<string, unknown>
  | string[]
  | null;

export type ApiEnvelope<T> = {
  data: T;
  message?: string;
};
