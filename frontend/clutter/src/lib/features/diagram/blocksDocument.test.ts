import { describe, expect, it } from "vitest";
import {
  BLOCK_NODE_TYPE,
  defaultBlockName,
  emptyBlocksDocument,
  fromBlocksDocument,
  isBlocksDocument,
  newBlockId,
  NODE_ID_PATTERN,
  toBlocksDocument,
  type BlockNode,
} from "./blocksDocument";

const node = (id: string, kind: string, name: string, fields: Record<string, unknown> = {}): BlockNode => ({
  id,
  type: BLOCK_NODE_TYPE,
  position: { x: 10, y: 20 },
  selected: true,
  measured: { width: 150, height: 50 },
  data: { kind, name, fields },
});

describe("toBlocksDocument", () => {
  it("writes the shape B's translator reads: engine, node type as kind, data.name and data.fields", () => {
    const doc = toBlocksDocument(
      [node("n1", "vm", "web", { size: "small" }), node("n2", "network", "net")],
      [{ id: "e1", source: "n1", target: "n2", animated: true }],
    );
    expect(doc).toEqual({
      engine: "blocks",
      nodes: [
        { id: "n1", type: "vm", position: { x: 10, y: 20 }, data: { name: "web", fields: { size: "small" } } },
        { id: "n2", type: "network", position: { x: 10, y: 20 }, data: { name: "net", fields: {} } },
      ],
      edges: [{ id: "e1", source: "n1", target: "n2" }],
    });
  });

  it("drops edges to missing blocks, self edges and repeated edges", () => {
    const doc = toBlocksDocument(
      [node("a", "vm", "a"), node("b", "network", "b")],
      [
        { id: "1", source: "a", target: "b" },
        { id: "2", source: "a", target: "b" },
        { id: "3", source: "a", target: "gone" },
        { id: "4", source: "a", target: "a" },
      ],
    );
    expect(doc.edges).toEqual([{ id: "1", source: "a", target: "b" }]);
  });

  it("does not share the fields object with the canvas", () => {
    const n = node("a", "vm", "a", { size: "small" });
    const doc = toBlocksDocument([n], []);
    doc.nodes[0].data.fields.size = "large";
    expect(n.data.fields.size).toBe("small");
  });
});

describe("fromBlocksDocument", () => {
  it("round-trips a saved document", () => {
    const nodes = [node("n1", "vm", "web", { size: "medium", public: true }), node("n2", "volume", "disk", { size_gb: 20 })];
    const edges = [{ id: "e1", source: "n2", target: "n1" }];
    const back = fromBlocksDocument(JSON.parse(JSON.stringify(toBlocksDocument(nodes, edges))));
    expect(back.nodes.map((n) => [n.id, n.type, n.data])).toEqual([
      ["n1", BLOCK_NODE_TYPE, { kind: "vm", name: "web", fields: { size: "medium", public: true } }],
      ["n2", BLOCK_NODE_TYPE, { kind: "volume", name: "disk", fields: { size_gb: 20 } }],
    ]);
    expect(back.edges).toEqual(edges);
  });

  it("skips what it cannot read instead of failing", () => {
    const back = fromBlocksDocument({
      engine: "blocks",
      nodes: [{ id: "ok", type: "vm", data: { name: 3 } }, { type: "vm" }, "junk", null],
      edges: [{ source: "ok", target: "missing" }, { source: "ok" }],
    });
    expect(back.nodes).toHaveLength(1);
    expect(back.nodes[0]).toMatchObject({ id: "ok", position: { x: 0, y: 0 }, data: { kind: "vm", name: "", fields: {} } });
    expect(back.edges).toEqual([]);
    expect(fromBlocksDocument(null)).toEqual({ nodes: [], edges: [] });
    expect(fromBlocksDocument("x")).toEqual({ nodes: [], edges: [] });
  });
});

describe("the engine marker", () => {
  it("marks a new diagram as building blocks, and only such documents", () => {
    expect(emptyBlocksDocument()).toEqual({ engine: "blocks", nodes: [], edges: [] });
    expect(isBlocksDocument(emptyBlocksDocument())).toBe(true);
    expect(isBlocksDocument({ nodes: [], edges: [] })).toBe(false);
    expect(isBlocksDocument({ engine: "aws" })).toBe(false);
    expect(isBlocksDocument(undefined)).toBe(false);
  });
});

describe("new blocks", () => {
  it("gets ids B accepts", () => {
    for (let i = 0; i < 20; i++) expect(newBlockId()).toMatch(NODE_ID_PATTERN);
  });

  it("gets a free, valid default name", () => {
    expect(defaultBlockName("vm", [])).toBe("vm");
    expect(defaultBlockName("vm", ["vm", "vm-2"])).toBe("vm-3");
    expect(defaultBlockName("http_api", [])).toBe("http_api");
    expect(defaultBlockName("9Weird Kind", [])).toBe("weird_kind");
  });
});
