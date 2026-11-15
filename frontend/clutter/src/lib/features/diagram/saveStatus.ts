// What the building-block editor shows beside Save, always: whether the canvas
// matches the saved diagram. A failed save keeps "Unsaved changes" (the
// failure itself is shown on its own line).
export type SaveStatus = { text: "Saving…" | "Unsaved changes" | "Saved"; tone: "busy" | "unsaved" | "saved" };

export function saveStatus({ dirty, saving }: { dirty: boolean; saving: boolean }): SaveStatus {
  if (saving) return { text: "Saving…", tone: "busy" };
  if (dirty) return { text: "Unsaved changes", tone: "unsaved" };
  return { text: "Saved", tone: "saved" };
}
