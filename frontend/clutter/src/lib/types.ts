import type { Connection, Edge, EdgeChange, Node, NodeChange, } from "@xyflow/react";

//store types 

//diagram types
export type DiagramNode = Node<NodeData>;
export type DiagramEdge = Edge;

export interface DiagramDataState {
  projectId: string | null;
  diagramId: string | null;
  nodes: DiagramNode[];
  edges: DiagramEdge[];
  isLoading: boolean;
  isSaving: boolean;
  dirty: boolean;
  error: string | null;
}

export interface DiagramActions {
  setContext: (projectId: string, diagramId: string) => void;
  setNodes: (nodes: DiagramNode[]) => void;
  setEdges: (edges: DiagramEdge[]) => void;
  updateNode: (nodeId: string, updates: Partial<DiagramNode>) => void;
  applyNodeChanges: (changes: NodeChange<DiagramNode>[]) => void;
  applyEdgeChanges: (changes: EdgeChange<DiagramEdge>[]) => void;
  addEdgeFromConnection: (conn: Connection) => void;
  addNode: (node: DiagramNode) => void;
  reset: () => void;
  loadDiagram: (projectId: string, diagramId: string) => Promise<void>;
  saveDiagram: () => Promise<void>;
}

export interface DiagramStore {
  state: DiagramDataState;
  actions: DiagramActions;
}

export type PaletteItem = {
  label: string;
  img: string;
};

export type NodeData = {
  label: string;
  img: string;
};
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