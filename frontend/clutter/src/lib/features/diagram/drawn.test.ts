import { describe, expect, it } from "vitest";
import { hasDrawnDiagram } from "./drawn";

const diagram = (nodes: number) => ({ data: { nodes: Array.from({ length: nodes }, (_, i) => ({ id: `n${i}` })) } });

describe("hasDrawnDiagram", () => {
  it("is true once any diagram in any project holds a block", () => {
    expect(hasDrawnDiagram([[diagram(0)], [diagram(0), diagram(2)]])).toBe(true);
  });

  it("is false for empty diagrams, no diagrams, or lists not loaded", () => {
    expect(hasDrawnDiagram([[diagram(0)], []])).toBe(false);
    expect(hasDrawnDiagram([])).toBe(false);
    expect(hasDrawnDiagram([undefined, null])).toBe(false);
    expect(hasDrawnDiagram([[{ data: null }, {}]])).toBe(false);
  });
});
