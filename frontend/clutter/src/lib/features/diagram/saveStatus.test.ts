import { describe, expect, it } from "vitest";
import { saveStatus } from "./saveStatus";

describe("saveStatus", () => {
  it("says Saved when the canvas matches the saved diagram", () => {
    expect(saveStatus({ dirty: false, saving: false })).toEqual({ text: "Saved", tone: "saved" });
  });

  it("says Unsaved changes while the canvas differs, including after a failed save", () => {
    expect(saveStatus({ dirty: true, saving: false })).toEqual({ text: "Unsaved changes", tone: "unsaved" });
  });

  it("says Saving while a save is on its way, whatever else is true", () => {
    expect(saveStatus({ dirty: true, saving: true }).text).toBe("Saving…");
    expect(saveStatus({ dirty: false, saving: true }).tone).toBe("busy");
  });
});
