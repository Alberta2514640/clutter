import { describe, expect, it } from "vitest";
import { levelLook } from "./types";

describe("levelLook", () => {
  it("gives each level its own label and a distinct tone", () => {
    const looks = (["starter", "intermediate", "advanced"] as const).map(levelLook);
    expect(looks.map((l) => l.label)).toEqual(["Starter", "Intermediate", "Advanced"]);
    expect(new Set(looks.map((l) => l.tone)).size).toBe(3);
    expect(looks.some((l) => l.tone === "neutral")).toBe(false);
  });

  it("shows a level it does not know by its own name, in the neutral tone", () => {
    expect(levelLook("expert")).toEqual({ label: "expert", tone: "neutral" });
  });
});
