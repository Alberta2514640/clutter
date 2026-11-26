// The undo history of B's canvas. Before each committed change (a block
// added or removed, a move when the drag ends, a line drawn or removed, a
// field edited, a box resized) the editor records the canvas as it was, so
// Ctrl or Cmd + Z puts the canvas back one change at a time, and Shift + Ctrl
// or Cmd + Z or Ctrl + Y does the change again. A removal's toast undoes
// back to before that removal.

// Relative imports: Vitest runs this file without the "@/" alias.
import type { BlockEdge, BlockNode } from "./blocksDocument";

export type CanvasState = { nodes: BlockNode[]; edges: BlockEdge[] };

// past holds the canvas before each change, newest last; future holds the
// canvas before each undo, newest last.
export type History = { past: CanvasState[]; future: CanvasState[] };

export const NO_HISTORY: History = { past: [], future: [] };

// The most changes the history keeps; the oldest go first.
export const HISTORY_LIMIT = 100;

type Step = { history: History; canvas: CanvasState };

// record keeps before, the canvas as it was before a change. A new change
// clears what was undone, as in any editor.
export const record = (h: History, before: CanvasState): History => ({ past: [...h.past, before].slice(-HISTORY_LIMIT), future: [] });

// undo is the canvas before the newest change, or null when there is none.
export function undo(h: History, now: CanvasState): Step | null {
  const prev = h.past[h.past.length - 1];
  if (!prev) return null;
  return { history: { past: h.past.slice(0, -1), future: [...h.future, now] }, canvas: prev };
}

// redo is the canvas the newest undo left, or null when there is none.
export function redo(h: History, now: CanvasState): Step | null {
  const next = h.future[h.future.length - 1];
  if (!next) return null;
  return { history: { past: [...h.past, now], future: h.future.slice(0, -1) }, canvas: next };
}

// undoTo undoes every change back to and including the one recorded with
// before (a removal's toast), so each can be done again in turn. It is null
// when that change is undone already or no longer kept.
export function undoTo(h: History, now: CanvasState, before: CanvasState): Step | null {
  const i = h.past.lastIndexOf(before);
  if (i < 0) return null;
  let step: Step = { history: h, canvas: now };
  while (step.history.past.length > i) step = undo(step.history, step.canvas)!;
  return step;
}

// removalLabel words what a removal took away for its toast: a block's name,
// "3 blocks", "a link" or "2 links". Lines that go with removed blocks are
// not counted.
export function removalLabel(nodes: Pick<BlockNode, "data">[], lines: number, titleOf: (kind: string) => string): string | null {
  if (nodes.length === 1) return nodes[0].data.name || titleOf(nodes[0].data.kind);
  if (nodes.length > 1) return `${nodes.length} blocks`;
  if (lines === 1) return "a link";
  if (lines > 1) return `${lines} links`;
  return null;
}

// The keys: Ctrl or Cmd + Z undoes, Shift + Ctrl or Cmd + Z and Ctrl + Y redo.
export type HistoryKey = { key: string; metaKey: boolean; ctrlKey: boolean; shiftKey: boolean; altKey: boolean };

export function historyShortcut(e: HistoryKey): "undo" | "redo" | null {
  if (!(e.metaKey || e.ctrlKey) || e.altKey) return null;
  const k = e.key.toLowerCase();
  if (k === "z") return e.shiftKey ? "redo" : "undo";
  if (k === "y" && e.ctrlKey && !e.metaKey && !e.shiftKey) return "redo";
  return null;
}

// Inputs that are not text: undo there still means the canvas.
const NOT_TEXT = new Set(["checkbox", "radio", "button", "submit", "reset", "range", "color", "file", "image"]);

// typingIn is whether a key press lands in a field that keeps its own undo
// (a text input, a text area, a select or editable text), so the canvas
// leaves the press alone.
export function typingIn(target: { tagName?: string; type?: string; isContentEditable?: boolean } | null): boolean {
  if (!target) return false;
  if (target.isContentEditable) return true;
  const tag = (target.tagName ?? "").toUpperCase();
  if (tag === "TEXTAREA" || tag === "SELECT") return true;
  return tag === "INPUT" && !NOT_TEXT.has((target.type ?? "text").toLowerCase());
}
