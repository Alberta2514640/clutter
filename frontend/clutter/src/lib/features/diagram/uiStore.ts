import type { DiagramEdge, DiagramNode } from "@/lib/features/diagram/types";
import { create } from "zustand";

const getStateHash = (name: string, nodes: DiagramNode[], edges: DiagramEdge[]): string => {
  return JSON.stringify({ name, nodes, edges });
};

type DiagramEditorSlice = {
  name: string;
  nodes: DiagramNode[];
  edges: DiagramEdge[];
  dirty: boolean;
  hydrated: boolean;
  serverHash: string; // hash of last-known server state (or last save)
};

type EditorState = {
  byId: Record<string, DiagramEditorSlice>;
};

type EditorActions = {
  ensure: (diagramId: string) => void;

  // now includes name
  hydrateFromServer: (diagramId: string, name: string, nodes: DiagramNode[], edges: DiagramEdge[]) => void;

  setName: (diagramId: string, name: string) => void;

  setNodes: (diagramId: string, nodes: DiagramNode[]) => void;
  setNodesWithoutDirty: (diagramId: string, nodes: DiagramNode[]) => void;

  setEdges: (diagramId: string, edges: DiagramEdge[]) => void;
  setEdgesWithoutDirty: (diagramId: string, edges: DiagramEdge[]) => void;

  markClean: (diagramId: string) => void;
  reset: (diagramId: string) => void;
};

const emptySlice = (): DiagramEditorSlice => ({
  name: "",
  nodes: [],
  edges: [],
  dirty: false,
  hydrated: false,
  serverHash: getStateHash("", [], []),
});

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
              [diagramId]: emptySlice(),
            },
          },
        };
      }),

    hydrateFromServer: (diagramId, name, nodes, edges) =>
      set((s) => {
        const cur = s.state.byId[diagramId];
        // If user already made edits, don't clobber them.
        if (cur?.dirty) return s;

        const serverHash = getStateHash(name, nodes, edges);

        return {
          state: {
            ...s.state,
            byId: {
              ...s.state.byId,
              [diagramId]: {
                name,
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

    setName: (diagramId, name) =>
      set((s) => {
        const cur = s.state.byId[diagramId] ?? emptySlice();
        const currentHash = getStateHash(name, cur.nodes, cur.edges);
        const dirty = currentHash !== cur.serverHash;

        return {
          state: {
            ...s.state,
            byId: {
              ...s.state.byId,
              [diagramId]: {
                ...cur,
                name,
                dirty,
              },
            },
          },
        };
      }),

    setNodes: (diagramId, nodes) =>
      set((s) => {
        const cur = s.state.byId[diagramId] ?? emptySlice();
        const currentHash = getStateHash(cur.name, nodes, cur.edges);
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
      set((s) => {
        const cur = s.state.byId[diagramId] ?? emptySlice();
        return {
          state: {
            ...s.state,
            byId: {
              ...s.state.byId,
              [diagramId]: { ...cur, nodes },
            },
          },
        };
      }),

    setEdges: (diagramId, edges) =>
      set((s) => {
        const cur = s.state.byId[diagramId] ?? emptySlice();
        const currentHash = getStateHash(cur.name, cur.nodes, edges);
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
      set((s) => {
        const cur = s.state.byId[diagramId] ?? emptySlice();
        return {
          state: {
            ...s.state,
            byId: {
              ...s.state.byId,
              [diagramId]: { ...cur, edges },
            },
          },
        };
      }),

    markClean: (diagramId) =>
      set((s) => {
        const cur = s.state.byId[diagramId];
        if (!cur) return s;

        const serverHash = getStateHash(cur.name, cur.nodes, cur.edges);

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
