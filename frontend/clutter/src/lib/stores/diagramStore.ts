// src/lib/stores/diagramStore.ts
import type { Connection, Edge, EdgeChange, Node, NodeChange } from "@xyflow/react";
import { addEdge, applyEdgeChanges, applyNodeChanges } from "@xyflow/react";
import { create } from "zustand";

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

const MOCK_NODES: DiagramNode[] = [
  {
    id: "912e7d9c-62ca-4416-af4c-da5826671e7d",
    type: "awsService",
    position: { x: 400, y: 280 },
    data: {
      label: "Lambda",
      img: "λ",
    },
  },
  {
    id: "3f9f1679-d083-4bdc-83de-aa72e5c9b148",
    type: "awsService",
    position: { x: 880, y: 280 },
    data: {
      label: "DynamoDB",
      img: "DB",
    },
  },
];

const MOCK_EDGES: DiagramEdge[] = [
  {
    id: "xy-edge__912e7d9c-62ca-4416-af4c-da5826671e7d-3f9f1679-d083-4bdc-83de-aa72e5c9b148",
    source: "912e7d9c-62ca-4416-af4c-da5826671e7d",
    target: "3f9f1679-d083-4bdc-83de-aa72e5c9b148",
  },
];

export const useDiagramStore = create<DiagramStore>((set, get) => ({
  state: {
    projectId: null,
    diagramId: null,
    nodes: [],
    edges: [],
    isLoading: false,
    isSaving: false,
    dirty: false,
    error: null,
  },

  actions: {
    setContext: (projectId, diagramId) =>
      set((s) => ({
        state: { ...s.state, projectId, diagramId },
      })),

    setNodes: (nodes) =>
      set((s) => {
        console.log("[diagramStore] setNodes -> mark dirty");
        return {
          state: { ...s.state, nodes, dirty: true },
        };
      }),

    setEdges: (edges) =>
      set((s) => ({
        state: { ...s.state, edges, dirty: true },
      })),

    updateNode: (nodeId: string, updates: Partial<DiagramNode>) =>
      set((s) => ({
        state: {
          ...s.state,
          nodes: s.state.nodes.map((node) => (node.id === nodeId ? { ...node, ...updates } : node)),
          dirty: true,
        },
      })),

    applyNodeChanges: (changes) =>
      set((s) => {
        // only want dirty in changes that the user makes that is add, remove and position.
        const shouldMarkDirty = changes.some(
          (change) => change.type === "position" || change.type === "add" || change.type === "remove"
        );
        return {
          state: {
            ...s.state,
            nodes: applyNodeChanges(changes, s.state.nodes),
            dirty: shouldMarkDirty ? true : s.state.dirty,
          },
        };
      }),

    applyEdgeChanges: (changes) =>
      set((s) => {
        const shouldMarkDirty = changes.some((change) => change.type === "add" || change.type === "remove");

        return {
          state: {
            ...s.state,
            edges: applyEdgeChanges(changes, s.state.edges),
            dirty: shouldMarkDirty ? true : s.state.dirty,
          },
        };
      }),

    addEdgeFromConnection: (conn) =>
      set((s) => ({
        state: {
          ...s.state,
          edges: addEdge({ ...conn }, s.state.edges),
          dirty: true,
        },
      })),

    addNode: (node) =>
      set((s) => ({
        state: {
          ...s.state,
          nodes: [...s.state.nodes, node],
          dirty: true,
        },
      })),

    reset: () =>
      set(() => ({
        state: {
          projectId: null,
          diagramId: null,
          nodes: [],
          edges: [],
          isLoading: false,
          isSaving: false,
          dirty: false,
          error: null,
        },
      })),

    loadDiagram: async (projectId, diagramId) => {
      set((s) => ({
        state: {
          ...s.state,
          projectId,
          diagramId,
          isLoading: true,
          error: null,
        },
      }));

      await new Promise((r) => setTimeout(r, 150));

      set((s) => ({
        state: {
          ...s.state,
          nodes: MOCK_NODES,
          edges: MOCK_EDGES,
          isLoading: false,
          dirty: false,
        },
      }));
    },

    saveDiagram: async () => {
      const { state } = get();
      const { projectId, diagramId, nodes, edges } = state;
      if (!projectId || !diagramId) return;

      set((s) => ({
        state: { ...s.state, isSaving: true, error: null },
      }));

      try {
        console.log("[diagramStore] saveDiagram", {
          projectId,
          diagramId,
          nodesCount: nodes.length,
          edgesCount: edges.length,
        });

        set((s) => ({
          state: { ...s.state, isSaving: false, dirty: false },
        }));
      } catch (e) {
        set((s) => ({
          state: { ...s.state, isSaving: false },
        }));
      }
    },
  },
}));

export const useDiagramState = () => useDiagramStore((s) => s.state);

export const useDiagramActions = () => useDiagramStore((s) => s.actions);
