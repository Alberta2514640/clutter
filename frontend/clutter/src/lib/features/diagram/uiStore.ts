import type { DiagramEdge, DiagramNode } from "@/lib/features/diagram/types";
import { create } from "zustand";

// Helper to create a hash of the current state
// Hash is used to compare the local changes and the server state to limit saving api calls
const getStateHash = (nodes: DiagramNode[], edges: DiagramEdge[]): string => {
  return JSON.stringify({ nodes, edges });
};

type EditorState = {
  byId: Record<
    string,
    {
      nodes: DiagramNode[];
      edges: DiagramEdge[];
      dirty: boolean;
      hydrated: boolean;
      serverHash: string;
    }
  >;
};

type EditorActions = {
  ensure: (diagramId: string) => void;
  hydrateFromServer: (diagramId: string, nodes: DiagramNode[], edges: DiagramEdge[]) => void;
  setNodes: (diagramId: string, nodes: DiagramNode[]) => void;
  setNodesWithoutDirty: (diagramId: string, nodes: DiagramNode[]) => void;
  setEdges: (diagramId: string, edges: DiagramEdge[]) => void;
  setEdgesWithoutDirty: (diagramId: string, edges: DiagramEdge[]) => void;
  markClean: (diagramId: string) => void;
  reset: (diagramId: string) => void;
};

export const useDiagramEditorStore = create<{ state: EditorState; actions: EditorActions }>((set) => ({
  state: { byId: {} },
  actions: {
    ensure: (diagramId) =>
      set((s) => {
        if (s.state.byId[diagramId]) return s;
        return {
          state: {
            ...s.state,
            byId: {
              ...s.state.byId,
              [diagramId]: {
                nodes: [],
                edges: [],
                dirty: false,
                hydrated: false,
                serverHash: getStateHash([], []),
              },
            },
          },
        };
      }),

    hydrateFromServer: (diagramId, nodes, edges) =>
      set((s) => {
        const cur = s.state.byId[diagramId];
        // If user already made edits, don't clobber them.
        if (cur?.dirty) return s;

        const serverHash = getStateHash(nodes, edges);

        return {
          state: {
            ...s.state,
            byId: {
              ...s.state.byId,
              [diagramId]: {
                nodes,
                edges,
                dirty: false,
                hydrated: true,
                serverHash,
              },
            },
          },
        };
      }),

    setNodes: (diagramId, nodes) =>
      set((s) => {
        const cur = s.state.byId[diagramId] ?? {
          nodes: [],
          edges: [],
          dirty: false,
          hydrated: false,
          serverHash: getStateHash([], []),
        };

        const currentHash = getStateHash(nodes, cur.edges);
        const dirty = currentHash !== cur.serverHash;

        return {
          state: {
            ...s.state,
            byId: {
              ...s.state.byId,
              [diagramId]: {
                ...cur,
                nodes,
                dirty,
              },
            },
          },
        };
      }),

    setNodesWithoutDirty: (diagramId, nodes) =>
      set((s) => ({
        state: {
          ...s.state,
          byId: {
            ...s.state.byId,
            [diagramId]: {
              ...(s.state.byId[diagramId] ?? {
                nodes: [],
                edges: [],
                dirty: false,
                hydrated: false,
                serverHash: getStateHash([], []),
              }),
              nodes,
              // Don't set dirty: true
            },
          },
        },
      })),

    setEdges: (diagramId, edges) =>
      set((s) => {
        const cur = s.state.byId[diagramId] ?? {
          nodes: [],
          edges: [],
          dirty: false,
          hydrated: false,
          serverHash: getStateHash([], []),
        };

        const currentHash = getStateHash(cur.nodes, edges);
        const dirty = currentHash !== cur.serverHash;

        return {
          state: {
            ...s.state,
            byId: {
              ...s.state.byId,
              [diagramId]: {
                ...cur,
                edges,
                dirty,
              },
            },
          },
        };
      }),

    setEdgesWithoutDirty: (diagramId, edges) =>
      set((s) => ({
        state: {
          ...s.state,
          byId: {
            ...s.state.byId,
            [diagramId]: {
              ...(s.state.byId[diagramId] ?? {
                nodes: [],
                edges: [],
                dirty: false,
                hydrated: false,
                serverHash: getStateHash([], []),
              }),
              edges,
              // Don't set dirty: true
            },
          },
        },
      })),

    markClean: (diagramId) =>
      set((s) => {
        const cur = s.state.byId[diagramId];
        if (!cur) return s;

        // Update server hash to current state after save
        const serverHash = getStateHash(cur.nodes, cur.edges);

        return {
          state: {
            ...s.state,
            byId: {
              ...s.state.byId,
              [diagramId]: {
                ...cur,
                dirty: false,
                serverHash,
              },
            },
          },
        };
      }),

    reset: (diagramId) =>
      set((s) => {
        const next = { ...s.state.byId };
        delete next[diagramId];
        return { state: { ...s.state, byId: next } };
      }),
  },
}));

export const useDiagramEditor = (diagramId: string) => useDiagramEditorStore((s) => s.state.byId[diagramId]);
export const useDiagramEditorActions = () => useDiagramEditorStore((s) => s.actions);
