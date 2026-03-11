import type { Edge, Node } from "@xyflow/react";

export type NodeData = { label: string; img: string };
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

  // backend GET uses "data"
  data: DiagramUiLayout;

  createdBy?: string;
  createdAt?: string;
  latestUpdateBy?: string;
  latestUpdateAt?: string;
};

/**  What the FRONTEND uses everywhere */
export type Diagram = {
  id: string;
  name: string;
  projectId: string;

  // frontend uses one consistent key
  uiLayout: DiagramUiLayout;

  createdBy?: string;
  createdAt?: string;
  latestUpdateBy?: string;
  latestUpdateAt?: string;
};

export type CreateDiagramInput = {
  projectId: string;
  name: string;
};

export type UpdateDiagramInput = {
  projectId: string;
  diagramId: string;
  name: string;
  uiLayout: DiagramUiLayout;
};

export type ApiEnvelope<T> = { data: T; message?: string };
