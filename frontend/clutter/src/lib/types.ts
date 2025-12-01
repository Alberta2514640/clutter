export interface CanvasBundle {
  canvas: Canvas;
  nodes: Node[];
  edges: Edge[];
  etag?: string;
  version?: number;
}
export interface Canvas {
  canvasId: string;
  name: string;
  uiLayout?: any;
}
export interface Node {
  nodeId: string;
  canvasId: string;
  resourceType: string;
  spec: any;
  iac?: { moduleAlias: string; moduleSource?: string; version?: string };
  ui?: any;
  etag?: string;
  version?: number;
}
export interface Edge {
  edgeId: string;
  fromNodeId: string;
  toNodeId: string;
  relation: string;
  props?: any;
  etag?: string;
  version?: number;
}
export interface Workspace {
  workspaceId: string;
  name: string;
  accountRef: { tenantId: string; accountId: string; alias: string };
  moduleSetId: string;
  defaultVarSetIds: string[];
}
export interface ModuleSet {
  moduleSetId: string;
  entries: { alias: string; source: string; version: string }[];
  artifactS3?: string;
}
export interface VarSet {
  varSetId: string;
  name: string;
  env: string;
  scope: "project" | "workspace";
  vars: Record<string, unknown>;
  secretRefs: { provider: "ssm" | "secretsmanager"; path?: string; arn?: string; envName: string }[];
}
export interface Run {
  runId: string;
  workspaceId: string;
  projectId: string;
  projectName: string;
  action: "plan" | "apply";
  status: RunStatus;
  startedAt: string;
  endedAt?: string;
  planS3?: string;
  applyLogS3?: string;
}

export type RunStatus = "QUEUED" | "INIT" | "PLAN" | "APPLY" | "FAILED" | "SUCCEEDED" | "CANCELED";
export interface User {
  userId: string;
  organizationId: string;
  email: string;
  displayName: string;
  pictureUrl: string;
  createdAt: string;
}