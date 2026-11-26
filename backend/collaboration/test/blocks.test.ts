import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import * as Y from "yjs";
import { canonical, fillDoc, parseSaved, readDoc, savedHash, type BlocksDocument } from "../src/blocks.ts";

const fixture = JSON.parse(readFileSync(new URL("./fixtures/blocks-doc.json", import.meta.url), "utf8"));

// Make a shared document from a saved one.
function filled(saved: unknown, name = "n"): Y.Doc {
  const ydoc = new Y.Doc();
  fillDoc(ydoc, saved, name);
  return ydoc;
}

// Read the three shared maps as plain JSON.
const sharedJson = (ydoc: Y.Doc) => ({
  nodes: ydoc.getMap("nodes").toJSON(),
  edges: ydoc.getMap("edges").toJSON(),
  meta: ydoc.getMap("meta").toJSON(),
});

// Copy every change from one document to another, like the network does.
function sync(a: Y.Doc, b: Y.Doc): void {
  Y.applyUpdate(b, Y.encodeStateAsUpdate(a));
  Y.applyUpdate(a, Y.encodeStateAsUpdate(b));
}

describe("the shared sample file", () => {
  it("fills the shared document as the sample says", () => {
    expect(sharedJson(filled(fixture.saved, fixture.name))).toEqual(fixture.shared);
  });

  it("reads the shared document back to the saved one", () => {
    const got = readDoc(filled(fixture.saved, fixture.name));
    expect(got).toEqual({ name: fixture.name, document: canonical(fixture.saved) });
  });

  it("gives the hash the sample says", () => {
    expect(savedHash(fixture.name, parseSaved(fixture.saved))).toBe(fixture.hash);
    const { name, document } = readDoc(filled(fixture.saved, fixture.name));
    expect(savedHash(name, document)).toBe(fixture.hash);
  });
});

describe("parseSaved", () => {
  it.each([null, undefined, "text", 5, [], {}])("gives an empty document for %j", (data) => {
    expect(parseSaved(data)).toEqual({ engine: "blocks", nodes: [], edges: [] });
  });

  it("skips bad nodes and fills missing values", () => {
    const got = parseSaved({
      nodes: [null, "x", { id: 1, type: "vm" }, { id: "a" }, { id: "b", type: "vm" }, { id: "c", type: "vm", position: { x: "1", y: 2 }, data: { name: 3, fields: [] } }],
    });
    expect(got.nodes).toEqual([
      { id: "b", type: "vm", position: { x: 0, y: 0 }, data: { name: "", fields: {} } },
      { id: "c", type: "vm", position: { x: 0, y: 2 }, data: { name: "", fields: {} } },
    ]);
  });

  it("skips edges to missing nodes and names an edge that has no id", () => {
    const nodes = [
      { id: "a", type: "vm" },
      { id: "b", type: "network" },
    ];
    const got = parseSaved({
      nodes,
      edges: [null, { source: "a" }, { id: "e1", source: "a", target: "gone" }, { source: "a", target: "b" }, { id: "", source: "b", target: "a" }],
    });
    expect(got.edges).toEqual([
      { id: "a-b", source: "a", target: "b" },
      { id: "b-a", source: "b", target: "a" },
    ]);
  });
});

describe("fillDoc", () => {
  it("stores each field as its own entry, so two users can edit two fields at once", () => {
    const server = filled(fixture.saved, fixture.name);
    const alice = new Y.Doc();
    const bob = new Y.Doc();
    sync(server, alice);
    sync(server, bob);

    const fieldsOf = (ydoc: Y.Doc) => (ydoc.getMap("nodes").get("node-vm") as Y.Map<unknown>).get("fields") as Y.Map<unknown>;
    expect(fieldsOf(alice)).toBeInstanceOf(Y.Map);
    fieldsOf(alice).set("flavor", "m1.large");
    fieldsOf(bob).set("startup_script", "echo hi");
    sync(alice, bob);

    expect(fieldsOf(alice).toJSON()).toMatchObject({ flavor: "m1.large", startup_script: "echo hi" });
    expect(fieldsOf(bob).toJSON()).toEqual(fieldsOf(alice).toJSON());
  });

  it("marks a new room as saved", () => {
    const ydoc = filled(fixture.saved, fixture.name);
    const { name, document } = readDoc(ydoc);
    expect(ydoc.getMap("meta").get("saved")).toEqual({ hash: savedHash(name, document), by: null, at: null });
  });
});

describe("readDoc", () => {
  it("removes bad edges that clients wrote", () => {
    const ydoc = filled(fixture.saved, fixture.name);
    const edges = ydoc.getMap("edges");
    edges.set("z-dangling", { source: "node-vm", target: "gone" });
    edges.set("z-self", { source: "node-vm", target: "node-vm" });
    edges.set("z-repeat", { source: "node-vm", target: "node-net" });
    edges.set("z-junk", "not an edge");
    expect(readDoc(ydoc).document.edges).toEqual(canonical(fixture.saved).edges);
  });

  it("skips nodes that are not valid", () => {
    const ydoc = filled(fixture.saved, fixture.name);
    const nodes = ydoc.getMap("nodes");
    nodes.set("junk-1", "text");
    nodes.set("junk-2", new Y.Map([["name", "no kind"]]));
    expect(readDoc(ydoc).document.nodes.map((n) => n.id)).toEqual(["node-net", "node-vm", "node-vol"]);
  });

  it("removes a node's edges from the result when the node is deleted", () => {
    const ydoc = filled(fixture.saved, fixture.name);
    ydoc.getMap("nodes").delete("node-vm");
    expect(readDoc(ydoc).document.edges).toEqual([]);
  });
});

describe("savedHash", () => {
  const doc = parseSaved(fixture.saved);

  it("does not depend on the order of nodes, edges or keys", () => {
    const shuffled: BlocksDocument = {
      engine: "blocks",
      edges: [...doc.edges].reverse(),
      nodes: [...doc.nodes].reverse().map((n) => ({ ...(n.width ? { height: n.height, width: n.width } : {}), data: { fields: n.data.fields, name: n.data.name }, position: n.position, type: n.type, id: n.id })),
    };
    expect(savedHash(fixture.name, shuffled)).toBe(fixture.hash);
  });

  it("changes when the name, a position, a field or a box size changes", () => {
    const moved = structuredClone(doc);
    moved.nodes[0]!.position.x += 1;
    const edited = structuredClone(doc);
    edited.nodes[0]!.data.fields.flavor = "m1.large";
    const resized = structuredClone(doc);
    Object.assign(resized.nodes.find((n) => n.id === "node-net")!, { width: 300, height: 160 });
    const hashes = [savedHash(fixture.name, doc), savedHash("Other", doc), savedHash(fixture.name, moved), savedHash(fixture.name, edited), savedHash(fixture.name, resized)];
    expect(new Set(hashes).size).toBe(5);
  });

  it("is the same as before for a diagram with no box sizes", () => {
    const plain = { engine: "blocks", nodes: [{ id: "a", type: "vm", position: { x: 1, y: 2 }, data: { name: "x", fields: {} } }], edges: [] };
    const withBadSize = { ...plain, nodes: [{ ...plain.nodes[0]!, width: 0, height: 10 }] };
    expect(savedHash("n", parseSaved(withBadSize))).toBe(savedHash("n", parseSaved(plain)));
  });
});

describe("box sizes", () => {
  const box = (width: unknown, height: unknown) => ({ nodes: [{ id: "net", type: "network", position: { x: 0, y: 0 }, width, height, data: { name: "n", fields: {} } }] });

  it("keeps a size when both width and height are positive numbers", () => {
    expect(parseSaved(box(260, 160)).nodes[0]).toMatchObject({ width: 260, height: 160 });
  });

  it.each([
    [260, undefined],
    [undefined, 160],
    [0, 160],
    [-5, 160],
    ["260", 160],
    [Number.NaN, 160],
    [Number.POSITIVE_INFINITY, 160],
  ])("drops the size for width %j and height %j", (w, h) => {
    const node = parseSaved(box(w, h)).nodes[0]!;
    expect("width" in node || "height" in node).toBe(false);
  });

  it("stores the size as one entry and reads it back", () => {
    const ydoc = filled(box(260, 160));
    const node = ydoc.getMap("nodes").get("net") as Y.Map<unknown>;
    expect(node.get("size")).toEqual({ width: 260, height: 160 });
    expect(readDoc(ydoc).document.nodes[0]).toMatchObject({ width: 260, height: 160 });
  });

  it("does not add a size to a node that has none", () => {
    const ydoc = filled(box(undefined, undefined));
    expect((ydoc.getMap("nodes").get("net") as Y.Map<unknown>).has("size")).toBe(false);
    expect("width" in readDoc(ydoc).document.nodes[0]!).toBe(false);
  });

  it("ignores a bad size that a client wrote", () => {
    const ydoc = filled(box(260, 160));
    (ydoc.getMap("nodes").get("net") as Y.Map<unknown>).set("size", { width: "wide", height: 160 });
    expect("width" in readDoc(ydoc).document.nodes[0]!).toBe(false);
  });

  it("keeps a resize when two users change the size and a field at once", () => {
    const a = filled(box(260, 160));
    const b = new Y.Doc();
    sync(a, b);
    (a.getMap("nodes").get("net") as Y.Map<unknown>).set("size", { width: 400, height: 200 });
    ((b.getMap("nodes").get("net") as Y.Map<unknown>).get("fields") as Y.Map<unknown>).set("cidr", "10.1.0.0/24");
    sync(a, b);
    expect(readDoc(b).document.nodes[0]).toMatchObject({ width: 400, height: 200, data: { fields: { cidr: "10.1.0.0/24" } } });
  });
});
