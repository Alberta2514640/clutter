import type { Edge, Node } from "@xyflow/react";

export type NodeData = {
  label: string;
  img: string;
  ansiblePlaybookName?: string;
  ansiblePlaybookKey?: string;
  ansiblePlaybookId?: string;
  ansibleTargetInstanceId?: string;
  lastAnsibleJobId?: string;
  lastAnsibleJobStatus?: string;
};
export type DiagramNode = Node<NodeData>;
export type DiagramEdge = Edge;

export type DiagramUiLayout = {
  nodes: DiagramNode[];
  edges: DiagramEdge[];
};

export type DiagramApiItem = {
  id: string;
  name: string;
  projectId: string;
  data: DiagramUiLayout;
  createdBy?: string;
  createdAt?: string;
  latestUpdateBy?: string;
  latestUpdateAt?: string;
};

// Diagram and DiagramApiItem now share the same shape
export type Diagram = DiagramApiItem;

export type CreateDiagramInput = {
  projectId: string;
  name: string;
  data?: DiagramUiLayout;
};

export type UpdateDiagramInput = {
  projectId: string;
  diagramId: string;
  name: string;
  data: DiagramUiLayout; // was uiLayout — now consistently "data"
};

export type ApiEnvelope<T> = { data: T; message?: string };
