// The building-block diagram document: what B's editor saves through A's
// diagram API into the diagrams table's data column. B owns its shape
// (backend/adapter/internal/translate):
//
//   { "engine": "blocks",
//     "nodes": [{ "id", "type": <block kind>, "position", "data": { "name", "fields" } }],
//     "edges": [{ "id", "source", "target" }] }
//
// A network node also keeps its box's "width" and "height", which B ignores.
//
// On the canvas a network is a box (ADR-025): a VM inside a box runs in that
// network, which means exactly the document's VM-to-Network edge. So the
// canvas holds that edge as containment, never as a line: reading a document
// puts each VM in the box of its first network edge, and saving writes the
// edge back. React Flow places a child relative to its parent; the document
// keeps every position on the canvas as a whole. These functions convert
// between the two shapes.

import type { Edge, Node, XYPosition } from "@xyflow/react";

export const BLOCKS_ENGINE = "blocks";

// The canvas node types: a box for a network, a card for every other block.
export const BLOCK_NODE_TYPE = "block";
export const NETWORK_NODE_TYPE = "network-box";

// The two kinds containment joins: a VM runs in a network.
export const NETWORK_KIND = "network";
export const VM_KIND = "vm";

// translate.NodeIDPattern: node ids address blocks in generated code.
export const NODE_ID_PATTERN = /^[A-Za-z0-9_-]{1,64}$/;

// The box layout. A block card is about BLOCK_W by BLOCK_H; a box holds its
// VMs in a column, PAD_X from its left side and PAD_TOP below its top, ROW apart.
export const BOX = {
  width: 260,
  height: 160,
  minWidth: 220,
  minHeight: 140,
  padX: 40,
  padTop: 50,
  padBottom: 36,
  row: 120,
  blockW: 180,
  blockH: 84,
} as const;

export type BlockNodeData = {
  kind: string;
  name: string;
  fields: Record<string, unknown>;
  // On a VM in a box: the id of the saved VM-to-Network edge the box stands
  // for, so a save keeps it. Never saved in data.
  networkEdgeId?: string;
};

export type BlockNode = Node<BlockNodeData, typeof BLOCK_NODE_TYPE | typeof NETWORK_NODE_TYPE>;
export type BlockEdge = Edge;

export type BlocksDocumentNode = {
  id: string;
  type: string;
  position: XYPosition;
  // A network's box size.
  width?: number;
  height?: number;
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
const positive = (v: unknown): number | null => (typeof v === "number" && Number.isFinite(v) && v > 0 ? v : null);

// nodeTypeFor is the canvas node type of a block kind.
export const nodeTypeFor = (kind: string) => (kind === NETWORK_KIND ? NETWORK_NODE_TYPE : BLOCK_NODE_TYPE);

// boxSize is a network node's box size on the canvas.
export function boxSize(n: BlockNode): { width: number; height: number } {
  return { width: n.width ?? n.measured?.width ?? BOX.width, height: n.height ?? n.measured?.height ?? BOX.height };
}

// absolutePosition is where a node sits on the canvas as a whole.
export function absolutePosition(n: BlockNode, byId: Map<string, BlockNode>): XYPosition {
  const parent = n.parentId ? byId.get(n.parentId) : undefined;
  return parent ? { x: parent.position.x + n.position.x, y: parent.position.y + n.position.y } : { x: n.position.x, y: n.position.y };
}

// parentsFirst puts every box before the other nodes, as React Flow needs a
// parent before its children; each group keeps its order.
export function parentsFirst(nodes: BlockNode[]): BlockNode[] {
  return [...nodes.filter((n) => n.type === NETWORK_NODE_TYPE), ...nodes.filter((n) => n.type !== NETWORK_NODE_TYPE)];
}

// toBlocksDocument builds the document to save from the canvas. Only the
// document's own keys are kept, positions are on the canvas as a whole, each
// VM in a box gets its VM-to-Network edge, and edges whose ends are missing
// are dropped.
export function toBlocksDocument(nodes: BlockNode[], edges: BlockEdge[]): BlocksDocument {
  const byId = new Map(nodes.map((n) => [n.id, n]));
  const seen = new Set<string>();
  const keep = (e: BlocksDocumentEdge): BlocksDocumentEdge[] => {
    const key = `${e.source}\u0000${e.target}`;
    if (!byId.has(e.source) || !byId.has(e.target) || e.source === e.target || seen.has(key)) return [];
    seen.add(key);
    return [e];
  };
  const contained = nodes.flatMap((n) => {
    const box = n.parentId ? byId.get(n.parentId) : undefined;
    if (!box || box.data.kind !== NETWORK_KIND || n.data.kind !== VM_KIND) return [];
    return keep({ id: n.data.networkEdgeId || `${n.id}-in-${box.id}`, source: n.id, target: box.id });
  });
  return {
    engine: BLOCKS_ENGINE,
    nodes: nodes.map((n) => {
      const out: BlocksDocumentNode = {
        id: n.id,
        type: n.data.kind,
        position: absolutePosition(n, byId),
        data: { name: n.data.name, fields: { ...n.data.fields } },
      };
      if (n.data.kind === NETWORK_KIND) {
        const s = boxSize(n);
        out.width = Math.round(s.width);
        out.height = Math.round(s.height);
      }
      return out;
    }),
    edges: [...contained, ...edges.flatMap((e) => keep({ id: e.id, source: e.source, target: e.target }))],
  };
}

// networksWithPrivateVM are the networks a VM that is not public runs in (its
// VM-to-Network edge, which a box on the canvas stands for).
export function networksWithPrivateVM(doc: BlocksDocument): Set<string> {
  const byId = new Map(doc.nodes.map((n) => [n.id, n]));
  const out = new Set<string>();
  for (const e of doc.edges) {
    const vm = byId.get(e.source);
    const net = byId.get(e.target);
    if (vm?.type === VM_KIND && net?.type === NETWORK_KIND && vm.data.fields.public !== true) out.add(net.id);
  }
  return out;
}

// fromBlocksDocument reads a saved document onto the canvas. It accepts any
// value, since the column holds whatever was last saved, and skips what it
// cannot read rather than failing the whole editor. A VM goes into the box
// of its first network edge; any other network edge of the VM stays a line,
// so nothing saved is lost. A box keeps its saved size and its VMs their
// places when they fit in it; otherwise the box is laid out around its VMs,
// in a column, at the network's place.
export function fromBlocksDocument(data: unknown): { nodes: BlockNode[]; edges: BlockEdge[] } {
  if (!isRecord(data)) return { nodes: [], edges: [] };
  const rawNodes = Array.isArray(data.nodes) ? data.nodes : [];
  const rawEdges = Array.isArray(data.edges) ? data.edges : [];

  const nodes: BlockNode[] = [];
  const saved = new Map<string, { width: number; height: number }>();
  for (const n of rawNodes) {
    if (!isRecord(n) || typeof n.id !== "string" || typeof n.type !== "string") continue;
    if (nodes.some((x) => x.id === n.id)) continue;
    const pos = isRecord(n.position) ? n.position : {};
    const d = isRecord(n.data) ? n.data : {};
    const node: BlockNode = {
      id: n.id,
      type: nodeTypeFor(n.type),
      position: { x: typeof pos.x === "number" ? pos.x : 0, y: typeof pos.y === "number" ? pos.y : 0 },
      data: {
        kind: n.type,
        name: typeof d.name === "string" ? d.name : "",
        fields: isRecord(d.fields) ? { ...d.fields } : {},
      },
    };
    if (n.type === NETWORK_KIND) {
      const w = positive(n.width);
      const h = positive(n.height);
      if (w && h) saved.set(n.id, { width: Math.max(w, BOX.minWidth), height: Math.max(h, BOX.minHeight) });
    }
    nodes.push(node);
  }
  const byId = new Map(nodes.map((n) => [n.id, n]));
  const edges: BlockEdge[] = [];
  for (const e of rawEdges) {
    if (!isRecord(e) || typeof e.source !== "string" || typeof e.target !== "string") continue;
    if (!byId.has(e.source) || !byId.has(e.target)) continue;
    edges.push({ id: typeof e.id === "string" && e.id ? e.id : `${e.source}-${e.target}`, source: e.source, target: e.target });
  }

  // Containment: each VM's first network edge becomes its box.
  const members = new Map<string, BlockNode[]>();
  const lines: BlockEdge[] = [];
  for (const e of edges) {
    const vm = byId.get(e.source)!;
    const net = byId.get(e.target)!;
    if (vm.data.kind === VM_KIND && net.data.kind === NETWORK_KIND && !vm.parentId && e.source !== e.target) {
      vm.parentId = net.id;
      vm.data.networkEdgeId = e.id;
      members.set(net.id, [...(members.get(net.id) ?? []), vm]);
      continue;
    }
    lines.push(e);
  }

  for (const net of nodes.filter((n) => n.data.kind === NETWORK_KIND)) {
    const inside = members.get(net.id) ?? [];
    const size = saved.get(net.id);
    const fits =
      !!size &&
      inside.every((vm) => {
        const x = vm.position.x - net.position.x;
        const y = vm.position.y - net.position.y;
        return x >= 0 && y >= 0 && x + BOX.blockW <= size.width + 1 && y + BOX.blockH <= size.height + 1;
      });
    if (fits) {
      net.width = size.width;
      net.height = size.height;
      for (const vm of inside) vm.position = { x: vm.position.x - net.position.x, y: vm.position.y - net.position.y };
      continue;
    }
    net.width = Math.max(size?.width ?? 0, BOX.width);
    net.height = Math.max(size?.height ?? 0, inside.length === 0 ? BOX.height : columnHeight(inside.length));
    inside.forEach((vm, i) => {
      vm.position = { x: BOX.padX, y: BOX.padTop + i * BOX.row };
    });
  }
  return { nodes: parentsFirst(nodes), edges: lines };
}

// columnHeight is the height of a box that holds n VMs in a column.
export const columnHeight = (n: number) => BOX.padTop + (Math.max(n, 1) - 1) * BOX.row + BOX.blockH + BOX.padBottom;

// boxAt is the box under a point on the canvas, or null: the last one drawn
// when boxes overlap, since it is on top.
export function boxAt(nodes: BlockNode[], point: XYPosition): string | null {
  let hit: string | null = null;
  for (const n of nodes) {
    if (n.type !== NETWORK_NODE_TYPE) continue;
    const { width, height } = boxSize(n);
    if (point.x >= n.position.x && point.x <= n.position.x + width && point.y >= n.position.y && point.y <= n.position.y + height) hit = n.id;
  }
  return hit;
}

// cardSize is a block card's size on the canvas, measured or assumed.
const cardSize = (n: BlockNode) => ({ w: n.measured?.width ?? BOX.blockW, h: n.measured?.height ?? BOX.blockH });

// placeIn puts a node whose top left corner is at topLeft on the canvas into
// the box boxId, or onto the canvas itself when boxId is null. In a box the
// node stays inside the box's padding and never covers another node of the
// box: when it would, it goes below them. The box grows to hold it. A node
// that changes box loses its saved edge id, so the save writes a new edge.
export function placeIn(nodes: BlockNode[], nodeId: string, boxId: string | null, topLeft: XYPosition): BlockNode[] {
  const box = boxId ? nodes.find((n) => n.id === boxId && n.type === NETWORK_NODE_TYPE) : undefined;
  const node = nodes.find((n) => n.id === nodeId);
  if (!node) return nodes;
  const changed = (node.parentId ?? null) !== (box?.id ?? null);
  const data = changed ? { ...node.data, networkEdgeId: undefined } : node.data;
  if (!box) {
    return parentsFirst(nodes.map((n) => (n.id === nodeId ? { ...unparented(n), data, position: { x: topLeft.x, y: topLeft.y } } : n)));
  }

  const { width, height } = boxSize(box);
  const size = cardSize(node);
  let x = Math.max(topLeft.x - box.position.x, 8);
  let y = Math.max(topLeft.y - box.position.y, BOX.padTop - 16);
  const others = nodes.filter((n) => n.parentId === box.id && n.id !== nodeId);
  const covers = (o: BlockNode) => {
    const s = cardSize(o);
    return x < o.position.x + s.w && x + size.w > o.position.x && y < o.position.y + s.h && y + size.h > o.position.y;
  };
  if (others.some(covers)) {
    x = BOX.padX;
    y = Math.max(...others.map((o) => o.position.y + cardSize(o).h)) + BOX.row - BOX.blockH;
  }
  const grown = { width: Math.max(width, x + size.w + 24), height: Math.max(height, y + size.h + BOX.padBottom) };
  return parentsFirst(
    nodes.map((n) => {
      if (n.id === nodeId) return { ...n, data, parentId: box.id, position: { x, y } };
      if (n.id === box.id) return { ...n, ...grown };
      return n;
    }),
  );
}

// detachFrom takes every node out of the given boxes and onto the canvas, at
// the same place, as when the boxes are removed.
export function detachFrom(nodes: BlockNode[], boxIds: Set<string>): BlockNode[] {
  const byId = new Map(nodes.map((n) => [n.id, n]));
  return nodes.map((n) => {
    if (!n.parentId || !boxIds.has(n.parentId)) return n;
    return { ...unparented(n), data: { ...n.data, networkEdgeId: undefined }, position: absolutePosition(n, byId) };
  });
}

// unparented is a copy of n that belongs to no box.
function unparented(n: BlockNode): BlockNode {
  const out = { ...n };
  delete out.parentId;
  delete out.extent;
  return out;
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
