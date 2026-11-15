// The building-block diagram document: what B's editor saves through A's
// diagram API into the diagrams table's data column. B owns its shape
// (backend/adapter/internal/translate):
//
//   { "engine": "blocks",
//     "nodes": [{ "id", "type": <block kind>, "position", "data": { "name", "fields" } }],
//     "edges": [{ "id", "source", "target" }] }
//
// On the canvas a block is a React Flow node of type "block" with its kind in
// data, so one node component draws every kind. These functions convert
// between the two shapes.

import type { Edge, Node, XYPosition } from "@xyflow/react";

export const BLOCKS_ENGINE = "blocks";

// The canvas node type for every block.
export const BLOCK_NODE_TYPE = "block";

// translate.NodeIDPattern: node ids address blocks in generated code.
export const NODE_ID_PATTERN = /^[A-Za-z0-9_-]{1,64}$/;

export type BlockNodeData = {
  kind: string;
  name: string;
  fields: Record<string, unknown>;
};

export type BlockNode = Node<BlockNodeData, typeof BLOCK_NODE_TYPE>;
export type BlockEdge = Edge;

export type BlocksDocumentNode = {
  id: string;
  type: string;
  position: XYPosition;
  data: { name: string; fields: Record<string, unknown> };
};

export type BlocksDocumentEdge = { id: string; source: string; target: string };

export type BlocksDocument = {
  engine: typeof BLOCKS_ENGINE;
  nodes: BlocksDocumentNode[];
  edges: BlocksDocumentEdge[];
};

// emptyBlocksDocument is what "New diagram" saves.
export const emptyBlocksDocument = (): BlocksDocument => ({ engine: BLOCKS_ENGINE, nodes: [], edges: [] });

// isBlocksDocument reports whether a saved document is B's, not A's.
export function isBlocksDocument(data: unknown): boolean {
  return !!data && typeof data === "object" && (data as { engine?: unknown }).engine === BLOCKS_ENGINE;
}

const isRecord = (v: unknown): v is Record<string, unknown> => !!v && typeof v === "object" && !Array.isArray(v);

// toBlocksDocument builds the document to save from the canvas. Only the
// document's own keys are kept, and edges whose ends are missing are dropped.
export function toBlocksDocument(nodes: BlockNode[], edges: BlockEdge[]): BlocksDocument {
  const ids = new Set(nodes.map((n) => n.id));
  const seen = new Set<string>();
  return {
    engine: BLOCKS_ENGINE,
    nodes: nodes.map((n) => ({
      id: n.id,
      type: n.data.kind,
      position: { x: n.position.x, y: n.position.y },
      data: { name: n.data.name, fields: { ...n.data.fields } },
    })),
    edges: edges.flatMap((e) => {
      const key = `${e.source}\u0000${e.target}`;
      if (!ids.has(e.source) || !ids.has(e.target) || e.source === e.target || seen.has(key)) return [];
      seen.add(key);
      return [{ id: e.id, source: e.source, target: e.target }];
    }),
  };
}

// fromBlocksDocument reads a saved document onto the canvas. It accepts any
// value, since the column holds whatever was last saved, and skips what it
// cannot read rather than failing the whole editor.
export function fromBlocksDocument(data: unknown): { nodes: BlockNode[]; edges: BlockEdge[] } {
  if (!isRecord(data)) return { nodes: [], edges: [] };
  const rawNodes = Array.isArray(data.nodes) ? data.nodes : [];
  const rawEdges = Array.isArray(data.edges) ? data.edges : [];

  const nodes: BlockNode[] = [];
  for (const n of rawNodes) {
    if (!isRecord(n) || typeof n.id !== "string" || typeof n.type !== "string") continue;
    const pos = isRecord(n.position) ? n.position : {};
    const d = isRecord(n.data) ? n.data : {};
    nodes.push({
      id: n.id,
      type: BLOCK_NODE_TYPE,
      position: { x: typeof pos.x === "number" ? pos.x : 0, y: typeof pos.y === "number" ? pos.y : 0 },
      data: {
        kind: n.type,
        name: typeof d.name === "string" ? d.name : "",
        fields: isRecord(d.fields) ? { ...d.fields } : {},
      },
    });
  }
  const ids = new Set(nodes.map((n) => n.id));
  const edges: BlockEdge[] = [];
  for (const e of rawEdges) {
    if (!isRecord(e) || typeof e.source !== "string" || typeof e.target !== "string") continue;
    if (!ids.has(e.source) || !ids.has(e.target)) continue;
    edges.push({ id: typeof e.id === "string" && e.id ? e.id : `${e.source}-${e.target}`, source: e.source, target: e.target });
  }
  return { nodes, edges };
}

// newBlockId returns a node id that fits NODE_ID_PATTERN.
export function newBlockId(): string {
  return crypto.randomUUID();
}

// defaultBlockName returns the kind itself, or the kind with the lowest free
// number, so every new block starts with a valid, distinct name.
export function defaultBlockName(kind: string, taken: string[]): string {
  const base = kind.toLowerCase().replace(/[^a-z0-9_-]/g, "_").replace(/^[^a-z]+/, "") || "block";
  const used = new Set(taken);
  if (!used.has(base)) return base;
  for (let i = 2; ; i++) {
    if (!used.has(`${base}-${i}`)) return `${base}-${i}`;
  }
}
