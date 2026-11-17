import { describe, expect, it } from "vitest";
import { diagramSummary } from "./summary";

const titleOf = (k: string) => ({ queue: "Queue", cache: "Cache" })[k] ?? k;

describe("diagramSummary", () => {
  it("counts each kind in plain words, in the standard order", () => {
    const s = diagramSummary(
      [
        { kind: "volume", name: "data" },
        { kind: "vm", name: "web" },
        { kind: "network", name: "app-net" },
        { kind: "vm", name: "db" },
        { kind: "http_api", name: "api" },
      ],
      titleOf,
    );
    expect(s.line).toBe("Builds 2 servers, 1 network, 1 disk, 1 API");
    expect(s.groups).toEqual([
      { kind: "vm", label: "2 servers", names: ["web", "db"] },
      { kind: "network", label: "1 network", names: ["app-net"] },
      { kind: "volume", label: "1 disk", names: ["data"] },
      { kind: "http_api", label: "1 API", names: ["api"] },
    ]);
  });

  it("names a cloud's own kind by its title, after the standard ones", () => {
    const s = diagramSummary(
      [
        { kind: "queue", name: "jobs" },
        { kind: "bucket", name: "" },
      ],
      titleOf,
    );
    expect(s.line).toBe("Builds 1 bucket, 1 Queue");
    expect(s.groups[0].names).toEqual(["(no name)"]);
  });

  it("says when there is nothing to build", () => {
    expect(diagramSummary([], titleOf)).toEqual({ line: "Nothing to build yet", groups: [] });
  });
});
