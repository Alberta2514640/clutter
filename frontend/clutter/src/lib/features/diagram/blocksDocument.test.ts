import { describe, expect, it } from "vitest";
import {
  BLOCK_NODE_TYPE,
  BOX,
  boxAt,
  columnHeight,
  defaultBlockName,
  detachFrom,
  fitBoxToChildren,
  NETWORK_NODE_TYPE,
  placeIn,
  emptyBlocksDocument,
  freeSpot,
  fromBlocksDocument,
  isBlocksDocument,
  networksWithVM,
  newBlockId,
  NODE_ID_PATTERN,
  toBlocksDocument,
  type BlockNode,
} from "./blocksDocument";

describe("networksWithVM", () => {
  it("names the networks a VM runs in, public or not", () => {
    const n = (id: string, type: string, fields: Record<string, unknown> = {}) => ({ id, type, position: { x: 0, y: 0 }, data: { name: id, fields } });
    const doc = {
      engine: "blocks" as const,
      nodes: [n("net-a", "network"), n("net-b", "network"), n("pub", "vm", { public: true }), n("priv", "vm", { public: false }), n("bare", "vm")],
      edges: [
        { id: "e1", source: "pub", target: "net-a" },
        { id: "e2", source: "priv", target: "net-b" },
        { id: "e3", source: "bare", target: "net-b" },
      ],
    };
    expect([...networksWithVM(doc)].sort()).toEqual(["net-a", "net-b"]);
    expect(networksWithVM({ ...doc, edges: [] }).size).toBe(0);
  });
});

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
      [node("n1", "vm", "web", { size: "small" }), node("n2", "volume", "disk")],
      [{ id: "e1", source: "n2", target: "n1", animated: true }],
    );
    expect(doc).toEqual({
      engine: "blocks",
      nodes: [
        { id: "n1", type: "vm", position: { x: 10, y: 20 }, data: { name: "web", fields: { size: "small" } } },
        { id: "n2", type: "volume", position: { x: 10, y: 20 }, data: { name: "disk", fields: {} } },
      ],
      edges: [{ id: "e1", source: "n2", target: "n1" }],
    });
  });

  it("drops edges to missing blocks, self edges and repeated edges", () => {
    const doc = toBlocksDocument(
      [node("a", "vm", "a"), node("b", "volume", "b")],
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

// A document as B saves it: a network with a VM inside its saved box, a VM
// with no box size saved (an older diagram), and a VM on two networks.
const saved = {
  engine: "blocks",
  nodes: [
    { id: "web", type: "vm", position: { x: 340, y: 50 }, data: { name: "web", fields: {} } },
    { id: "app-net", type: "network", position: { x: 300, y: 0 }, width: 260, height: 160, data: { name: "app-net", fields: { cidr: "10.0.0.0/24" } } },
    { id: "fw", type: "firewall", position: { x: 0, y: 30 }, data: { name: "fw", fields: {} } },
  ],
  edges: [
    { id: "web-on-net", source: "web", target: "app-net" },
    { id: "fw-on-web", source: "fw", target: "web" },
  ],
};

describe("networks as boxes", () => {
  it("puts a VM in the box of its network edge, keeps its place inside a saved box, and puts boxes first", () => {
    const { nodes, edges } = fromBlocksDocument(saved);
    expect(nodes.map((n) => [n.id, n.type])).toEqual([
      ["app-net", NETWORK_NODE_TYPE],
      ["web", BLOCK_NODE_TYPE],
      ["fw", BLOCK_NODE_TYPE],
    ]);
    const web = nodes.find((n) => n.id === "web")!;
    expect(web.parentId).toBe("app-net");
    expect(web.position).toEqual({ x: 40, y: 50 });
    expect(nodes[0]).toMatchObject({ width: 260, height: 160 });
    // The network edge is the box, not a line.
    expect(edges).toEqual([{ id: "fw-on-web", source: "fw", target: "web" }]);
  });

  it("saves the box as the VM-to-Network edge, with its id, and every position on the canvas as a whole", () => {
    const { nodes, edges } = fromBlocksDocument(saved);
    const doc = toBlocksDocument(nodes, edges);
    expect(doc.edges).toEqual([
      { id: "web-on-net", source: "web", target: "app-net" },
      { id: "fw-on-web", source: "fw", target: "web" },
    ]);
    expect(doc.nodes.find((n) => n.id === "web")?.position).toEqual({ x: 340, y: 50 });
    expect(doc.nodes.find((n) => n.id === "app-net")).toMatchObject({ width: 260, height: 160 });
    // Read again, it is the same canvas.
    const again = fromBlocksDocument(JSON.parse(JSON.stringify(doc)));
    expect(toBlocksDocument(again.nodes, again.edges)).toEqual(doc);
  });

  it("lays a box out around its VMs when the document has no box size or a VM outside it", () => {
    const { nodes } = fromBlocksDocument({
      engine: "blocks",
      nodes: [
        { id: "a", type: "vm", position: { x: 900, y: 900 }, data: { name: "a", fields: {} } },
        { id: "b", type: "vm", position: { x: -50, y: 0 }, data: { name: "b", fields: {} } },
        { id: "net", type: "network", position: { x: 100, y: 100 }, data: { name: "net", fields: {} } },
        { id: "empty", type: "network", position: { x: 0, y: 0 }, data: { name: "empty", fields: {} } },
      ],
      edges: [
        { id: "1", source: "a", target: "net" },
        { id: "2", source: "b", target: "net" },
      ],
    });
    const byId = new Map(nodes.map((n) => [n.id, n]));
    expect(byId.get("net")).toMatchObject({ position: { x: 100, y: 100 }, width: BOX.width, height: columnHeight(2) });
    expect(byId.get("a")).toMatchObject({ parentId: "net", position: { x: BOX.padX, y: BOX.padTop } });
    expect(byId.get("b")).toMatchObject({ parentId: "net", position: { x: BOX.padX, y: BOX.padTop + BOX.row } });
    expect(byId.get("empty")).toMatchObject({ width: BOX.width, height: BOX.height });
  });

  it("keeps a VM's second network edge as a line, so nothing saved is lost", () => {
    const { nodes, edges } = fromBlocksDocument({
      engine: "blocks",
      nodes: [
        { id: "vm", type: "vm", position: { x: 0, y: 0 }, data: { name: "vm", fields: {} } },
        { id: "n1", type: "network", position: { x: 0, y: 0 }, data: { name: "n1", fields: {} } },
        { id: "n2", type: "network", position: { x: 400, y: 0 }, data: { name: "n2", fields: {} } },
      ],
      edges: [
        { id: "first", source: "vm", target: "n1" },
        { id: "second", source: "vm", target: "n2" },
      ],
    });
    expect(nodes.find((n) => n.id === "vm")?.parentId).toBe("n1");
    expect(edges).toEqual([{ id: "second", source: "vm", target: "n2" }]);
    expect(toBlocksDocument(nodes, edges).edges.map((e) => e.id)).toEqual(["first", "second"]);
  });

  it("finds the box under a point, places a VM in it, inside its padding, and takes it out again", () => {
    const { nodes } = fromBlocksDocument(saved);
    expect(boxAt(nodes, { x: 310, y: 10 })).toBe("app-net");
    expect(boxAt(nodes, { x: 10, y: 10 })).toBeNull();

    // A VM dropped at the box's edge stays where it was dropped, and the box
    // grows to hold it.
    const extra: BlockNode = { id: "db", type: BLOCK_NODE_TYPE, position: { x: 0, y: 400 }, data: { kind: "vm", name: "db", fields: {} } };
    let next = placeIn([...nodes, extra], "db", "app-net", { x: 555, y: 155 });
    const db = next.find((n) => n.id === "db")!;
    expect(db.parentId).toBe("app-net");
    expect(db.position).toEqual({ x: 255, y: 155 });
    expect(next.find((n) => n.id === "app-net")).toMatchObject({ width: 255 + BOX.blockW + 24, height: 155 + BOX.blockH + BOX.padBottom });
    expect(toBlocksDocument(next, []).edges).toContainEqual({ id: "db-in-app-net", source: "db", target: "app-net" });

    // Dropped on top of the VM already in the box, it moves to a free spot
    // inside the box's edges, and the box grows to hold it.
    const onTop = placeIn([...nodes, extra], "db", "app-net", { x: 340, y: 50 });
    const moved = onTop.find((n) => n.id === "db")!;
    const sibling = onTop.find((n) => n.id === "web")!;
    const overlaps =
      moved.position.x < sibling.position.x + BOX.blockW &&
      moved.position.x + BOX.blockW > sibling.position.x &&
      moved.position.y < sibling.position.y + BOX.blockH &&
      moved.position.y + BOX.blockH > sibling.position.y;
    expect(overlaps).toBe(false);
    expect(moved.position.x).toBeGreaterThanOrEqual(BOX.edgeLeft);
    expect(moved.position.y).toBeGreaterThanOrEqual(BOX.edgeTop);
    const grown = onTop.find((n) => n.id === "app-net")!;
    expect(grown.width).toBeGreaterThanOrEqual(moved.position.x + BOX.blockW + BOX.edgeRight);
    expect(grown.height).toBeGreaterThanOrEqual(moved.position.y + BOX.blockH + BOX.padBottom);

    // Moved out, it is on the canvas where it was dropped, and its edge is gone.
    next = placeIn(next, "web", null, { x: 700, y: 20 });
    const web = next.find((n) => n.id === "web")!;
    expect(web.parentId).toBeUndefined();
    expect(web.position).toEqual({ x: 700, y: 20 });
    expect(toBlocksDocument(next, []).edges.map((e) => e.source)).toEqual(["db"]);
  });

  it("leaves a removed box's VMs on the canvas where they were", () => {
    const { nodes } = fromBlocksDocument(saved);
    const left = detachFrom(nodes, new Set(["app-net"])).filter((n) => n.id !== "app-net");
    expect(left.find((n) => n.id === "web")).toMatchObject({ position: { x: 340, y: 50 } });
    expect(left.find((n) => n.id === "web")?.parentId).toBeUndefined();
    expect(toBlocksDocument(left, []).edges).toEqual([]);
  });
});

describe("fitBoxToChildren", () => {
  const box = (w: number, h: number): BlockNode => ({
    id: "net",
    type: NETWORK_NODE_TYPE,
    position: { x: 100, y: 100 },
    width: w,
    height: h,
    data: { kind: "network", name: "net", fields: {} },
  });
  const vm = (x: number, y: number): BlockNode => ({ id: "vm", type: BLOCK_NODE_TYPE, parentId: "net", position: { x, y }, data: { kind: "vm", name: "vm", fields: {} } });

  it("grows a box made smaller than its VM back around it", () => {
    const out = fitBoxToChildren([box(150, 100), vm(40, 50)], "net");
    expect(out[0]).toMatchObject({ position: { x: 100, y: 100 }, width: 40 + BOX.blockW + BOX.edgeRight, height: 50 + BOX.blockH + BOX.padBottom });
    expect(out[1].position).toEqual({ x: 40, y: 50 });
  });

  it("moves a box pulled past its VM from the left or top back out, leaving the VM where it was", () => {
    // The VM sits at 90,110 on the canvas; the box's left side was pulled to its right.
    const out = fitBoxToChildren([box(300, 300), vm(-10, 10)], "net");
    const [b, v] = out;
    expect(b.position).toEqual({ x: 100 - 10 - BOX.edgeLeft, y: 100 + 10 - BOX.edgeTop });
    expect(v.position).toEqual({ x: BOX.edgeLeft, y: BOX.edgeTop });
    expect(b.position.x + v.position.x).toBe(90);
    expect(b.position.y + v.position.y).toBe(110);
    expect(b.width).toBe(300 + 10 + BOX.edgeLeft);
  });

  it("leaves a box that holds its VMs, and an empty box, as they are", () => {
    const nodes = [box(400, 300), vm(40, 50)];
    expect(fitBoxToChildren(nodes, "net")).toBe(nodes);
    const empty = [box(100, 100)];
    expect(fitBoxToChildren(empty, "net")).toBe(empty);
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

describe("freeSpot", () => {
  const card = (id: string, x: number, y: number): BlockNode => ({ id, type: BLOCK_NODE_TYPE, position: { x, y }, data: { kind: "vm", name: id, fields: {} } });
  const size = { w: BOX.blockW, h: BOX.blockH };
  const overlaps = (p: { x: number; y: number }, n: BlockNode) =>
    p.x < n.position.x + BOX.blockW && p.x + size.w > n.position.x && p.y < n.position.y + BOX.blockH && p.y + size.h > n.position.y;

  it("centres a card on an empty canvas", () => {
    expect(freeSpot([], { x: 400, y: 300 }, size)).toEqual({ x: 320, y: 260 });
  });

  it("never stacks a new card on the ones already there", () => {
    const nodes: BlockNode[] = [];
    for (let i = 0; i < 6; i++) {
      const p = freeSpot(nodes, { x: 400, y: 300 }, size);
      for (const n of nodes) expect(overlaps(p, n)).toBe(false);
      nodes.push(card(`n${i}`, p.x, p.y));
    }
  });

  it("never goes left of or above the given corner", () => {
    const p = freeSpot([card("a", 0, 40)], { x: 90, y: 80 }, size, { x: BOX.edgeLeft, y: BOX.edgeTop });
    expect(p.x).toBeGreaterThanOrEqual(BOX.edgeLeft);
    expect(p.y).toBeGreaterThanOrEqual(BOX.edgeTop);
    expect(overlaps(p, card("a", 0, 40))).toBe(false);
  });

  it("keeps clear of a network box", () => {
    const box: BlockNode = { id: "net", type: NETWORK_NODE_TYPE, position: { x: 200, y: 200 }, width: 400, height: 200, data: { kind: "network", name: "net", fields: {} } };
    const p = freeSpot([box], { x: 400, y: 300 }, size);
    const inside = p.x < 600 && p.x + size.w > 200 && p.y < 400 && p.y + size.h > 200;
    expect(inside).toBe(false);
  });
});
