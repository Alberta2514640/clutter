import { describe, expect, it } from "vitest";
import { DIAGRAM_NAME_MAX, uniqueDiagramName } from "./diagramName";

describe("uniqueDiagramName", () => {
  it("keeps a free name", () => {
    expect(uniqueDiagramName("Web server", ["New Diagram"])).toBe("Web server");
  });

  it("numbers a taken name from 2", () => {
    expect(uniqueDiagramName("Web server", ["Web server"])).toBe("Web server 2");
    expect(uniqueDiagramName("Web server", ["Web server", "Web server 2"])).toBe("Web server 3");
  });

  it("fits the cap with its number", () => {
    const long = "A".repeat(40);
    const first = uniqueDiagramName(long, []);
    expect(first).toHaveLength(DIAGRAM_NAME_MAX);
    const second = uniqueDiagramName(long, [first]);
    expect(second.length).toBeLessThanOrEqual(DIAGRAM_NAME_MAX);
    expect(second.endsWith(" 2")).toBe(true);
  });
});
