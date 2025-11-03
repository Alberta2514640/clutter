import { create } from "zustand";
import type { CanvasBundle, Node, Edge } from "@/lib/types";
type EditorState = {
  bundle?: CanvasBundle;
  selectedIds: string[];
  dirty: boolean;
  setBundle: (b: CanvasBundle) => void;
  upsertNode: (n: Node) => void;
  upsertEdge: (e: Edge) => void;
  removeById: (id: string) => void;
  markClean: () => void;
};
export const useEditor = create<EditorState>()((set, get) => ({
  selectedIds: [],
  dirty: false,
  setBundle: (b) => set({ bundle: b, dirty: false }),
  upsertNode: (n) => set((s) => ({ bundle: s.bundle ? { ...s.bundle, nodes: upsert(s.bundle.nodes, n) } : s.bundle, dirty: true })),
  upsertEdge: (e) => set((s) => ({ bundle: s.bundle ? { ...s.bundle, edges: upsert(s.bundle.edges, e) } : s.bundle, dirty: true })),
  removeById: (id) => set((s) => (s.bundle ? { bundle: { ...s.bundle, nodes: s.bundle.nodes.filter((n) => n.nodeId !== id), edges: s.bundle.edges.filter((x) => x.edgeId !== id && x.fromNodeId !== id && x.toNodeId !== id) }, dirty: true } : s)),
  markClean: () => set({ dirty: false }),
}));
const upsert = <T extends { [k: string]: any }>(arr: T[], item: T) => {
  const key = (item as any).nodeId ?? (item as any).edgeId;
  const idx = arr.findIndex((x) => ((x as any).nodeId ?? (x as any).edgeId) === key);
  return idx >= 0 ? arr.map((x, i) => (i === idx ? item : x)) : [...arr, item];
};
