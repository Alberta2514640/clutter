// src/lib/stores/diagramStore.ts
import { addEdge, applyEdgeChanges, applyNodeChanges, type Connection, } from "@xyflow/react";
import { create } from "zustand";

import type { DiagramEdge, DiagramNode, DiagramState } from "@/lib/types";

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
    type: "default",
    style: {
      stroke: "rgba(100,180,255,0.6)",
      strokeWidth: 2,
    },
  },
];

export const useDiagramStore = create<DiagramState>((set, get) => ({
  projectId: null,
  diagramId: null,

  nodes: [],
  edges: [],

  isLoading: false,
  isSaving: false,
  dirty: false,
  error: null,

  setContext: (projectId, diagramId) => set({ projectId, diagramId }),

  setNodes: (nodes: DiagramNode[]) =>
    set({
      nodes,
      dirty: true,
    }),

  setEdges: (edges: DiagramEdge[]) =>
    set({
      edges,
      dirty: true,
    }),

  applyNodeChanges: (changes) =>
    set((state) => ({
      nodes: applyNodeChanges<DiagramNode>(changes, state.nodes),
      dirty: true,
    })),

  applyEdgeChanges: (changes) =>
    set((state) => ({
      edges: applyEdgeChanges(changes, state.edges),
      dirty: true,
    })),

  addEdgeFromConnection: (params: Connection) =>
    set((state) => ({
      edges: addEdge(
        {
          ...params,
          type: "default",
          style: { stroke: "rgba(100,180,255,0.6)", strokeWidth: 2 },
        },
        state.edges
      ),
      dirty: true,
    })),

  addNode: (node: DiagramNode) =>
    set((state) => ({
      nodes: [...state.nodes, node],
      dirty: true,
    })),

  reset: () =>
    set({
      projectId: null,
      diagramId: null,
      nodes: [],
      edges: [],
      isLoading: false,
      isSaving: false,
      dirty: false,
      error: null,
    }),

  // --------- PLACEHOLDER: load from API later ---------
  loadDiagram: async (projectId: string, diagramId: string) => {
    set({
      projectId,
      diagramId,
      isLoading: true,
      error: null,
    });

    // fake a tiny delay so you can see loading behavior if you want
    await new Promise((r) => setTimeout(r, 150));

    console.log("[diagramStore] loadDiagram (mock)", {
      projectId,
      diagramId,
      nodes: MOCK_NODES,
      edges: MOCK_EDGES,
    });

    // For now: always load the same mock diagram
    set({
      nodes: MOCK_NODES,
      edges: MOCK_EDGES,
      isLoading: false,
      dirty: false,
    });
  },


  // --------- PLACEHOLDER: save to API later ---------
  saveDiagram: async () => {
    const { projectId, diagramId, nodes, edges } = get();
    if (!projectId || !diagramId) {
      console.warn("[diagramStore] saveDiagram called without context");
      return;
    }

    set({ isSaving: true, error: null });

    try {
      // TODO: replace with real fetch to your API
      console.log("[diagramStore] saveDiagram (placeholder)", {
        projectId,
        diagramId,
        nodesCount: nodes.length,
        edgesCount: edges.length,
      });

      set({
        isSaving: false,
        dirty: false,
      });
    } catch (err: unknown) {
      set({
        isSaving: false,
      });
    }
  },
}));
