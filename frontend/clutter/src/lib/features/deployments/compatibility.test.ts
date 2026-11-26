import { describe, expect, it } from "vitest";
import type { Manifest } from "../catalog/types";
import { blockIssues, compatibilityLine, nextExclusions, problemsByBlock, splitReadyLine, unsupportedBlocks, type BlockRef } from "./compatibility";
import type { Compatibility, Problem } from "./types";

const blocks: BlockRef[] = [
  { id: "n1", kind: "vm", name: "web" },
  { id: "n2", kind: "network", name: "net" },
  { id: "n3", kind: "bucket", name: "files" },
  { id: "n4", kind: "function", name: "thumbs" },
];

const titles: Record<string, string> = { vm: "VM", network: "Network", bucket: "Bucket", function: "Function" };
const kindTitle = (k: string) => titles[k] ?? k;

const unsupported = (id: string, title: string): Problem => ({
  node_id: id,
  code: "unsupported_block",
  message: `OpenStack cannot build a ${title} block. Remove it from the diagram or leave it out of this deployment.`,
});

describe("compatibilityLine", () => {
  it("says a cloud that builds everything can build every block, not that anything is deployed", () => {
    const line = compatibilityLine("AWS", { ready: true, problems: [], notes: [] }, blocks, [], kindTitle);
    expect(line).toEqual({ ready: true, summary: "AWS: every block can be built here", choices: [], problems: [], notes: [] });
  });

  it("on one cloud of a split, says only its placed blocks can be built, and one line says it for all clouds", () => {
    const os = compatibilityLine("Local OpenStack", { ready: true, problems: [], notes: [] }, blocks.slice(0, 2), [], kindTitle, true);
    const aws = compatibilityLine("Local AWS", { ready: true, problems: [], notes: [] }, blocks.slice(2, 3), [], kindTitle, true);
    expect(os.summary).toBe("Local OpenStack: the 2 blocks placed here can be built");
    expect(aws.summary).toBe("Local AWS: the 1 block placed here can be built");
    expect(splitReadyLine([os, aws], ["Local OpenStack", "Local AWS"], 0)).toBe("Every block can be built across Local OpenStack and Local AWS");
    expect(splitReadyLine([os, aws], ["Local OpenStack", "Local AWS"], 1)).toBe("The other blocks can be built across Local OpenStack and Local AWS, 1 block left out");
    const broken = compatibilityLine("Local AWS", { ready: false, problems: [unsupported("n3", "Bucket")], notes: [] }, blocks.slice(2, 3), [], kindTitle, true);
    expect(splitReadyLine([os, broken], ["Local OpenStack", "Local AWS"], 0)).toBeNull();
    expect(splitReadyLine([os], ["Local OpenStack"], 0)).toBeNull();
  });

  it("names the unsupported blocks and offers to leave each out", () => {
    const report: Compatibility = { ready: false, problems: [unsupported("n3", "Bucket"), unsupported("n4", "Function")], notes: [] };
    const line = compatibilityLine("OpenStack", report, blocks, [], kindTitle);
    expect(line.ready).toBe(false);
    expect(line.summary).toBe("OpenStack: 2 blocks not supported (Bucket, Function)");
    expect(line.choices).toEqual([
      { id: "n3", name: "files", kindTitle: "Bucket", supported: false, excluded: false },
      { id: "n4", name: "thumbs", kindTitle: "Function", supported: false, excluded: false },
    ]);
    expect(line.problems).toEqual([]);
  });

  it("counts left-out blocks once ready, and keeps them toggleable", () => {
    const line = compatibilityLine("OpenStack", { ready: true, problems: null, notes: null }, blocks, ["n3", "n4"], kindTitle);
    expect(line.summary).toBe("OpenStack: the other blocks can be built here, 2 blocks left out");
    expect(line.choices.map((c) => [c.id, c.excluded, c.supported])).toEqual([
      ["n3", true, true],
      ["n4", true, true],
    ]);
  });

  it("words other problems and notes with the block's name, and ignores exclusions of removed blocks", () => {
    const report: Compatibility = {
      ready: false,
      problems: [
        unsupported("n3", "Bucket"),
        { node_id: "n1", field: "size", code: "invalid_value", message: "Size must be one of the listed values." },
        { edge_source: "n1", edge_target: "n2", code: "unsupported_edge", message: "OpenStack cannot connect a VM to a Network." },
        { field: "image", code: "missing_value", message: "Image is required." },
      ],
      notes: [{ node_id: "gone", code: "unknown_exclusion", message: "This left-out block is no longer in the diagram." }],
    };
    const line = compatibilityLine("OpenStack", report, blocks, ["gone"], kindTitle);
    expect(line.summary).toBe("OpenStack: 1 block not supported (Bucket), 3 other problems");
    expect(line.problems).toEqual([
      "web: Size must be one of the listed values.",
      "web to net: OpenStack cannot connect a VM to a Network.",
      "Settings: Image is required.",
    ]);
    expect(line.notes).toEqual(["gone: This left-out block is no longer in the diagram."]);
    expect(line.choices.map((c) => c.id)).toEqual(["n3"]);
  });

  it("says problem, not other problem, when no block is unsupported", () => {
    const report: Compatibility = { ready: false, problems: [{ field: "image", code: "missing_value", message: "Image is required." }], notes: [] };
    expect(compatibilityLine("OpenStack", report, blocks, [], kindTitle).summary).toBe("OpenStack: 1 problem");
  });
});

describe("before a deployment exists", () => {
  const openstack: Manifest = {
    connector: "general-openstack",
    title: "OpenStack",
    version: "1",
    blocks: [
      { kind: "vm", treatment: "as-is", title: "VM" },
      { kind: "network", treatment: "as-is", title: "Network" },
      { kind: "bucket", treatment: "removed", title: "Bucket" },
    ],
    edges: null,
    settings: null,
    display: { short_name: "OpenStack", logo: "logo" },
  };

  it("lists the blocks a cloud's manifest does not build", () => {
    expect(unsupportedBlocks(openstack, blocks).map((b) => b.id)).toEqual(["n3", "n4"]);
  });
});

describe("nextExclusions", () => {
  it("adds and removes one block and drops blocks no longer in the diagram", () => {
    expect(nextExclusions([], blocks, "n3", true)).toEqual(["n3"]);
    expect(nextExclusions(["n3", "gone"], blocks, "n4", true)).toEqual(["n3", "n4"]);
    expect(nextExclusions(["n3", "n4"], blocks, "n3", false)).toEqual(["n4"]);
    expect(nextExclusions(["n3"], blocks, "n3", true)).toEqual(["n3"]);
    expect(nextExclusions([], blocks, "missing", true)).toEqual([]);
  });
});

describe("problemsByBlock", () => {
  it("puts each problem on its block, or on the block its line starts from, once, and skips notes", () => {
    const report: Compatibility = {
      ready: false,
      problems: [
        unsupported("n3", "Bucket"),
        unsupported("n3", "Bucket"),
        { edge_source: "n1", edge_target: "n2", code: "invalid_edge", message: "On AWS a VM runs in one network. Connect it to only one." },
        { field: "image", code: "missing_value", message: "Settings: the image is required." },
      ],
      notes: [{ node_id: "n4", code: "note", message: "A note." }],
    };
    const p = problemsByBlock(report);
    expect([...p.keys()]).toEqual(["n3", "n1"]);
    expect(p.get("n3")).toHaveLength(1);
    expect(p.get("n1")).toEqual(["On AWS a VM runs in one network. Connect it to only one."]);
    expect(problemsByBlock(undefined).size).toBe(0);
  });
});

describe("blockIssues", () => {
  it("puts each problem under its field, keeps the block's others and its notes, and skips cost notes", () => {
    const report: Compatibility = {
      ready: false,
      problems: [
        { node_id: "fw", field: "ingress", code: "invalid_value", message: "Each allowed entry must look like tcp/22." },
        { node_id: "fw", field: "ingress", code: "invalid_value", message: "Each allowed entry must look like tcp/22." },
        { node_id: "fw", field: "unknown", code: "unknown_field", message: "This field is not used by this cloud." },
        { edge_source: "fw", edge_target: "vm", code: "unsupported_edge", message: "This cloud cannot connect them." },
        { node_id: "vm", field: "size", code: "invalid_value", message: "Not this one." },
      ],
      notes: [
        { node_id: "vm", code: "public_vm_closed", message: "This VM is public but its firewall allows nothing in." },
        { node_id: "vm", code: "cost", message: "Costs money." },
      ],
    };
    const fw = blockIssues(report, "fw", new Set(["ingress", "source_cidr"]));
    expect([...fw.fields]).toEqual([["ingress", ["Each allowed entry must look like tcp/22."]]]);
    expect(fw.general).toEqual(["This field is not used by this cloud.", "This cloud cannot connect them."]);
    expect(blockIssues(report, "vm", new Set(["size"])).notes).toEqual(["This VM is public but its firewall allows nothing in."]);
    expect(blockIssues(undefined, "vm", new Set()).general).toEqual([]);
  });
});
