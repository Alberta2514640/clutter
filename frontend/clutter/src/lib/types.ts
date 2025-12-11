import type { Connection, Edge, EdgeChange, Node, NodeChange, } from "@xyflow/react";

//store types 

//diagram types
export type DiagramNode = Node<NodeData>;
export type DiagramEdge = Edge;

export type DiagramState = {
  projectId: string | null;
  diagramId: string | null;

  nodes: DiagramNode[];
  edges: DiagramEdge[];

  // NEW flags
  isLoading: boolean;
  isSaving: boolean;
  dirty: boolean;
  error: string | null;

  setContext: (projectId: string, diagramId: string) => void;

  setNodes: (nodes: DiagramNode[]) => void;
  setEdges: (edges: DiagramEdge[]) => void;

  applyNodeChanges: (changes: NodeChange<DiagramNode>[]) => void;
  applyEdgeChanges: (changes: EdgeChange[]) => void;

  addEdgeFromConnection: (params: Connection) => void;
  addNode: (node: DiagramNode) => void;

  reset: () => void;

  // NEW fake persistence actions
  loadDiagram: (projectId: string, diagramId: string) => Promise<void>;
  saveDiagram: () => Promise<void>;
};

export type PaletteItem = {
  label: string;
  img: string;
};

export type NodeData = {
  label: string;
  img: string;
};


//this was here before idk
export type RunStatus = "QUEUED" | "INIT" | "PLAN" | "APPLY" | "FAILED" | "SUCCEEDED" | "CANCELED";
export interface Run {
  runId: string;
  workspaceId: string;
  action: "plan" | "apply";
  status: RunStatus;
  startedAt?: string;
  endedAt?: string;
  planS3?: string;
  applyLogS3?: string;
}
