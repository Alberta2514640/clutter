import { describe, expect, it } from "vitest";
import { BLOCK_NODE_TYPE, type BlockNode } from "./blocksDocument";
import { HISTORY_LIMIT, historyShortcut, NO_HISTORY, record, redo, removalLabel, typingIn, undo, undoTo, type CanvasState } from "./history";

const block = (id: string, x = 0): BlockNode => ({ id, type: BLOCK_NODE_TYPE, position: { x, y: 0 }, data: { kind: "vm", name: id, fields: {} } });
const canvas = (...nodes: BlockNode[]): CanvasState => ({ nodes, edges: [] });

describe("the canvas history", () => {
  const a = canvas();
  const b = canvas(block("web"));
  const c = canvas(block("web", 200));

  it("undoes the newest change first, then redoes it", () => {
    // Two changes: add web (a to b), move it (b to c).
    const h = record(record(NO_HISTORY, a), b);
    const u1 = undo(h, c)!;
    expect(u1.canvas).toBe(b);
    const u2 = undo(u1.history, u1.canvas)!;
    expect(u2.canvas).toBe(a);
    expect(undo(u2.history, u2.canvas)).toBeNull();
    const r1 = redo(u2.history, u2.canvas)!;
    expect(r1.canvas).toBe(b);
    const r2 = redo(r1.history, r1.canvas)!;
    expect(r2.canvas).toBe(c);
    expect(redo(r2.history, r2.canvas)).toBeNull();
  });

  it("clears what was undone when a new change is made", () => {
    const h = record(record(NO_HISTORY, a), b);
    const u = undo(h, c)!;
    expect(record(u.history, u.canvas).future).toEqual([]);
  });

  it("keeps only the newest changes", () => {
    let h = NO_HISTORY;
    for (let i = 0; i < HISTORY_LIMIT + 5; i++) h = record(h, canvas(block(`n${i}`)));
    expect(h.past).toHaveLength(HISTORY_LIMIT);
    expect(h.past[0].nodes[0].id).toBe("n5");
  });

  it("lets a toast undo back to before its own removal, once, and redo each change after", () => {
    // b is the canvas before the removal; a later move followed (c).
    const h = record(record(NO_HISTORY, a), b);
    const later = canvas(block("db"));
    const step = undoTo(h, later, b)!;
    expect(step.canvas).toBe(b);
    expect(step.history.past).toEqual([a]);
    expect(undoTo(step.history, step.canvas, b)).toBeNull();
    expect(redo(step.history, step.canvas)!.canvas).toBe(later);
  });
});

describe("removalLabel", () => {
  const title = (k: string) => (k === "network" ? "Network" : k);

  it("names one block, counts several, and falls back to the kind", () => {
    expect(removalLabel([block("web")], 2, title)).toBe("web");
    expect(removalLabel([block("web"), block("db")], 0, title)).toBe("2 blocks");
    expect(removalLabel([{ data: { kind: "network", name: "", fields: {} } }], 0, title)).toBe("Network");
  });

  it("words lines when no block is removed", () => {
    expect(removalLabel([], 1, title)).toBe("a link");
    expect(removalLabel([], 2, title)).toBe("2 links");
    expect(removalLabel([], 0, title)).toBeNull();
  });
});

describe("historyShortcut and typingIn", () => {
  const key = (k: string, mods: Partial<{ metaKey: boolean; ctrlKey: boolean; shiftKey: boolean; altKey: boolean }> = {}) => ({
    key: k,
    metaKey: false,
    ctrlKey: false,
    shiftKey: false,
    altKey: false,
    ...mods,
  });

  it("reads undo and redo on Mac and elsewhere", () => {
    expect(historyShortcut(key("z", { metaKey: true }))).toBe("undo");
    expect(historyShortcut(key("z", { ctrlKey: true }))).toBe("undo");
    expect(historyShortcut(key("Z", { metaKey: true, shiftKey: true }))).toBe("redo");
    expect(historyShortcut(key("y", { ctrlKey: true }))).toBe("redo");
    expect(historyShortcut(key("y", { metaKey: true }))).toBeNull();
    expect(historyShortcut(key("z"))).toBeNull();
    expect(historyShortcut(key("z", { ctrlKey: true, altKey: true }))).toBeNull();
  });

  it("leaves text fields their own undo, but not a checkbox or the canvas", () => {
    expect(typingIn({ tagName: "INPUT", type: "text" })).toBe(true);
    expect(typingIn({ tagName: "input" })).toBe(true);
    expect(typingIn({ tagName: "TEXTAREA" })).toBe(true);
    expect(typingIn({ tagName: "DIV", isContentEditable: true })).toBe(true);
    expect(typingIn({ tagName: "INPUT", type: "checkbox" })).toBe(false);
    expect(typingIn({ tagName: "DIV" })).toBe(false);
    expect(typingIn(null)).toBe(false);
  });
});
