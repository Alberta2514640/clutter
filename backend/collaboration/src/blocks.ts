// Converts a saved blocks diagram to the shared Yjs document, and back.
// The saved format belongs to B (frontend blocksDocument.ts). We do not change it.
// The frontend has its own copy of this conversion.
// Both copies are tested with test/fixtures/blocks-doc.json so they stay the same.

import { createHash } from "node:crypto";
import * as Y from "yjs";

export interface BlocksNode {
  id: string;
  type: string;
  position: { x: number; y: number };
  data: { name: string; fields: Record<string, unknown> };
}

export interface BlocksEdge {
  id: string;
  source: string;
  target: string;
}

export interface BlocksDocument {
  engine: "blocks";
  nodes: BlocksNode[];
  edges: BlocksEdge[];
}

const isRecord = (v: unknown): v is Record<string, unknown> => !!v && typeof v === "object" && !Array.isArray(v);

// Compare two ids as plain strings. Do not use localeCompare here.
// It can give a different order on another machine.
const byId = (a: { id: string }, b: { id: string }): number => (a.id < b.id ? -1 : a.id > b.id ? 1 : 0);

// Read a saved document. It can hold anything, so we check every value.
// Bad nodes and edges are skipped. This follows fromBlocksDocument in the frontend.
export function parseSaved(data: unknown): BlocksDocument {
  const doc: BlocksDocument = { engine: "blocks", nodes: [], edges: [] };
  if (!isRecord(data)) return doc;
  const rawNodes = Array.isArray(data.nodes) ? data.nodes : [];
  const rawEdges = Array.isArray(data.edges) ? data.edges : [];

  for (const n of rawNodes) {
    if (!isRecord(n) || typeof n.id !== "string" || typeof n.type !== "string") continue;
    const pos = isRecord(n.position) ? n.position : {};
    const d = isRecord(n.data) ? n.data : {};
    doc.nodes.push({
      id: n.id,
      type: n.type,
      position: { x: typeof pos.x === "number" ? pos.x : 0, y: typeof pos.y === "number" ? pos.y : 0 },
      data: {
        name: typeof d.name === "string" ? d.name : "",
        fields: isRecord(d.fields) ? { ...d.fields } : {},
      },
    });
  }

  const ids = new Set(doc.nodes.map((n) => n.id));
  for (const e of rawEdges) {
    if (!isRecord(e) || typeof e.source !== "string" || typeof e.target !== "string") continue;
    if (!ids.has(e.source) || !ids.has(e.target)) continue;
    const id = typeof e.id === "string" && e.id ? e.id : `${e.source}-${e.target}`;
    doc.edges.push({ id, source: e.source, target: e.target });
  }
  return doc;
}

// Put a document in a fixed order, so the same diagram always looks the same.
// Nodes and edges are sorted by id. Bad edges are removed:
// an edge to a missing node, an edge from a node to itself, and a repeated edge.
// This follows toBlocksDocument in the frontend.
export function canonical(doc: BlocksDocument): BlocksDocument {
  const nodes = [...doc.nodes].sort(byId);
  const ids = new Set(nodes.map((n) => n.id));
  const seen = new Set<string>();
  const edges: BlocksEdge[] = [];
  for (const e of [...doc.edges].sort(byId)) {
    const key = `${e.source}\u0000${e.target}`;
    if (!ids.has(e.source) || !ids.has(e.target) || e.source === e.target || seen.has(key)) continue;
    seen.add(key);
    edges.push(e);
  }
  return { engine: "blocks", nodes, edges };
}

// Turn a value into JSON text with object keys in sorted order.
function stableJson(value: unknown): string {
  if (Array.isArray(value)) return `[${value.map(stableJson).join(",")}]`;
  if (isRecord(value)) {
    const keys = Object.keys(value).sort();
    return `{${keys.map((k) => `${JSON.stringify(k)}:${stableJson(value[k])}`).join(",")}}`;
  }
  return JSON.stringify(value) ?? "null";
}

// The hash of a diagram's name and content. The editor uses it to show "Saved".
// Two diagrams with the same content always get the same hash.
export function savedHash(name: string, doc: BlocksDocument): string {
  const { nodes, edges } = canonical(doc);
  return createHash("sha256").update(stableJson({ name, nodes, edges })).digest("hex");
}

// Fill an empty shared document from the saved copy. Only the server does this.
// Layout:
//   nodes: map of node id -> map { kind, name, position, fields (a map) }
//   edges: map of edge id -> { source, target }
//   meta:  map { name, saved: { hash, by, at } }
export function fillDoc(ydoc: Y.Doc, saved: unknown, name: string): void {
  const doc = parseSaved(saved);
  ydoc.transact(() => {
    const nodes = ydoc.getMap("nodes");
    for (const n of doc.nodes) {
      // Each field is its own entry. Two users can then edit two fields at once.
      const fields = new Y.Map<unknown>();
      for (const [key, value] of Object.entries(n.data.fields)) fields.set(key, value);
      const node = new Y.Map<unknown>();
      node.set("kind", n.type);
      node.set("name", n.data.name);
      node.set("position", { x: n.position.x, y: n.position.y });
      node.set("fields", fields);
      nodes.set(n.id, node);
    }

    const edges = ydoc.getMap("edges");
    for (const e of doc.edges) edges.set(e.id, { source: e.source, target: e.target });

    // The room starts equal to the saved copy, so it counts as saved.
    const meta = ydoc.getMap("meta");
    meta.set("name", name);
    meta.set("saved", { hash: savedHash(name, doc), by: null, at: null });
  });
}

// Read the shared document back into the saved format.
// Clients can write anything, so we check every value again.
export function readDoc(ydoc: Y.Doc): { name: string; document: BlocksDocument } {
  const nodes: BlocksNode[] = [];
  ydoc.getMap("nodes").forEach((node, id) => {
    if (!(node instanceof Y.Map)) return;
    const kind = node.get("kind");
    if (typeof kind !== "string") return;
    const name = node.get("name");
    const pos = node.get("position");
    const fields = node.get("fields");
    nodes.push({
      id,
      type: kind,
      position: {
        x: isRecord(pos) && typeof pos.x === "number" ? pos.x : 0,
        y: isRecord(pos) && typeof pos.y === "number" ? pos.y : 0,
      },
      data: {
        name: typeof name === "string" ? name : "",
        fields: fields instanceof Y.Map ? (fields.toJSON() as Record<string, unknown>) : {},
      },
    });
  });

  const edges: BlocksEdge[] = [];
  ydoc.getMap("edges").forEach((edge, id) => {
    if (!isRecord(edge) || typeof edge.source !== "string" || typeof edge.target !== "string") return;
    edges.push({ id, source: edge.source, target: edge.target });
  });

  const name = ydoc.getMap("meta").get("name");
  return { name: typeof name === "string" ? name : "", document: canonical({ engine: "blocks", nodes, edges }) };
}
