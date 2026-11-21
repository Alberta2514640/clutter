import { describe, expect, it } from "vitest";
import { diagramSummary } from "./summary";

const titleOf = (k: string) => ({ queue: "Queue", cache: "Cache" })[k] ?? k;
const deployment = (excluded: string[] = []) => ({ excluded });

describe("diagramSummary", () => {
  it("counts each kind in plain words, in the standard order", () => {
    const s = diagramSummary(
      [
        { id: "d", kind: "volume", name: "data" },
        { id: "w", kind: "vm", name: "web" },
        { id: "n", kind: "network", name: "app-net" },
        { id: "b", kind: "vm", name: "db" },
        { id: "a", kind: "http_api", name: "api" },
      ],
      titleOf,
      deployment(),
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
      deployment(),
    );
    expect(s.line).toBe("Builds 1 bucket, 1 Queue");
    expect(s.groups[0].names).toEqual(["(no name)"]);
  });

  it("never counts a block the deployment leaves out as built", () => {
    const blocks = [
      { id: "w", kind: "vm", name: "web" },
      { id: "f", kind: "bucket", name: "files" },
    ];
    const s = diagramSummary(blocks, titleOf, deployment(["f"]));
    expect(s.line).toBe("Builds 1 server, 1 left out");
    expect(s.groups.map((g) => g.kind)).toEqual(["vm"]);
    expect(diagramSummary(blocks, titleOf, deployment(["w", "f"])).line).toBe("Builds nothing, 2 left out");
  });

  it("says what the diagram has when no deployment is chosen", () => {
    const s = diagramSummary([{ id: "w", kind: "vm", name: "web" }], titleOf);
    expect(s.line).toBe("Has 1 block");
    expect(s.groups).toHaveLength(1);
  });

  it("says when there is nothing to build", () => {
    expect(diagramSummary([], titleOf)).toEqual({ line: "Nothing to build yet", groups: [] });
    expect(diagramSummary([], titleOf, deployment())).toEqual({ line: "Nothing to build yet", groups: [] });
  });
});
