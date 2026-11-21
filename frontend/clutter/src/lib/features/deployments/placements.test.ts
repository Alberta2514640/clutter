import { describe, expect, it } from "vitest";
import type { Connection } from "../connections/types";
import {
  blockCloud,
  cloudBlocks,
  cloudChoices,
  cloudsLine,
  crossNotes,
  deploymentTitle,
  examplePresets,
  exampleSplitFor,
  missingExampleClouds,
  movePlacement,
  needsTwoClouds,
  orderedClouds,
  partLines,
  placeBlock,
  placementReason,
  placementSuggestions,
  splitOnly,
  suggestedPlacements,
  withPlacementSettings,
} from "./placements";
import type { Compatibility, Placement } from "./types";

const blocks = [
  { id: "web", kind: "vm", name: "web" },
  { id: "db", kind: "vm", name: "db" },
  { id: "files", kind: "bucket", name: "files" },
];

const lab: Placement = { connection_id: "c-lab", connection_name: "lab", connector: "general-openstack", region: "RegionOne", node_ids: ["db"], settings: { image: "cirros" } };

const conn = (id: string, connector: string, status: Connection["status"] = "working"): Connection => ({
  id,
  organization_id: "o",
  connector,
  name: id,
  region: "r",
  status,
  status_detail: null,
  created_at: "",
  updated_at: "",
  verified_at: null,
});

describe("example presets", () => {
  const hybrid = {
    name: "Hybrid",
    clouds: ["general-openstack", "aws"],
    placements: [{ connector: "aws", node_ids: ["files"] }],
    document: { nodes: [{ id: "web", type: "vm" }, { id: "files", type: "bucket" }] },
  };
  const single = { name: "Web server", clouds: ["aws", "general-openstack"], document: { nodes: [{ id: "files", type: "bucket" }] } };
  const offers = (connector: string, kind: string) => connector === "aws" || kind !== "bucket";

  it("finds the split example a diagram came from by its placed blocks' ids and kinds", () => {
    const split = exampleSplitFor([single, hybrid], blocks);
    expect(split?.primary).toBe("general-openstack");
    expect([...(split?.placed ?? [])]).toEqual([["files", "aws"]]);
    expect(exampleSplitFor([hybrid], [{ id: "files", kind: "vm", name: "files" }])).toBeNull();
    expect(exampleSplitFor([hybrid], [{ id: "x1", kind: "bucket", name: "files" }])).toBeNull();
  });

  it("names the example's clouds the team has no working connection to", () => {
    const split = exampleSplitFor([hybrid], blocks);
    expect(missingExampleClouds(split, [conn("a", "aws")])).toEqual(["general-openstack"]);
    expect(missingExampleClouds(split, [conn("a", "aws"), conn("o", "general-openstack", "failed")])).toEqual(["general-openstack"]);
    expect(missingExampleClouds(split, [conn("o", "general-openstack")])).toEqual(["aws"]);
    expect(missingExampleClouds(split, [conn("a", "aws"), conn("o", "general-openstack")])).toEqual([]);
    expect(missingExampleClouds(null, [])).toEqual([]);
  });

  // The examples as B serves them (backend/adapter/internal/examples): node
  // ids and kinds only, which is all the match reads.
  const doc = (nodes: [string, string][]) => ({ nodes: nodes.map(([id, type]) => ({ id, type })) });
  const asBlocks = (nodes: [string, string][]) => nodes.map(([id, kind]) => ({ id, kind, name: id }));
  const fileStorageNodes: [string, string][] = [["files", "bucket"]];
  const fullAppNodes: [string, string][] = [
    ["web-fw", "firewall"], ["web", "vm"], ["files", "bucket"], ["db-fw", "firewall"], ["db-data", "volume"],
    ["db", "vm"], ["app-net", "network"], ["orders-api", "http_api"], ["orders", "function"], ["orders-table", "table"],
  ];
  const hybridNodes: [string, string][] = [["app-fw", "firewall"], ["app", "vm"], ["app-net", "network"], ["files", "bucket"]];
  const served = [
    { name: "File storage", clouds: ["aws"], document: doc(fileStorageNodes) },
    { name: "Full app", clouds: ["aws"], document: doc(fullAppNodes) },
    { name: "Hybrid", clouds: ["general-openstack", "aws"], placements: [{ connector: "aws", node_ids: ["files"] }], document: doc(hybridNodes) },
  ];

  it("matches the whole example document, so Full app and File storage are not Hybrid", () => {
    expect(exampleSplitFor(served, asBlocks(fullAppNodes))).toBeNull();
    expect(exampleSplitFor(served, asBlocks(fileStorageNodes))).toBeNull();
  });

  it("matches a diagram made from Hybrid, also after the user adds a block", () => {
    const split = exampleSplitFor(served, asBlocks(hybridNodes));
    expect(split?.name).toBe("Hybrid");
    expect(split?.primary).toBe("general-openstack");
    expect([...(split?.placed ?? [])]).toEqual([["files", "aws"]]);
    expect(exampleSplitFor(served, [...asBlocks(hybridNodes), { id: "n-1a2b", kind: "table", name: "orders" }])?.name).toBe("Hybrid");
  });

  it("does not match Hybrid once one of its blocks is gone or changed kind", () => {
    expect(exampleSplitFor(served, asBlocks(hybridNodes.filter(([id]) => id !== "app-fw")))).toBeNull();
    expect(exampleSplitFor(served, asBlocks(hybridNodes.map(([id, k]) => [id, id === "app" ? "function" : k] as [string, string])))).toBeNull();
  });

  it("does not match a hand-drawn diagram", () => {
    const drawn = asBlocks([["n-7f3c", "vm"], ["n-91d0", "bucket"], ["n-44aa", "network"], ["n-0b1e", "firewall"]]);
    expect(exampleSplitFor(served, drawn)).toBeNull();
  });

  it("puts each placed block on a working connection of its cloud, only when the deployment is on the first cloud", () => {
    const split = exampleSplitFor([hybrid], blocks);
    const connections = [conn("c-os", "general-openstack"), conn("c-broken", "aws", "failed"), conn("c-aws", "aws")];
    const presets = examplePresets(split, connections[0], connections, offers, blocks);
    expect([...presets].map(([id, c]) => [id, c.id])).toEqual([["files", "c-aws"]]);
    expect(examplePresets(split, connections[2], connections, offers, blocks).size).toBe(0);
    expect(examplePresets(null, connections[0], connections, offers, blocks).size).toBe(0);
  });
});

describe("blockCloud", () => {
  it("is the placement's connection, or the deployment's own", () => {
    expect(blockCloud("c-aws", [lab], "db")).toBe("c-lab");
    expect(blockCloud("c-aws", [lab], "web")).toBe("c-aws");
    expect(blockCloud("c-aws", [], "db")).toBe("c-aws");
  });
});

describe("cloudsLine", () => {
  const titleOf = (c: string) => ({ "general-openstack": "OpenStack", aws: "Amazon Web Services" })[c] ?? c;

  it("names one cloud and its region", () => {
    expect(cloudsLine({ title: "Amazon Web Services", region: "ca-central-1" }, null, titleOf)).toBe("Amazon Web Services, ca-central-1");
    expect(cloudsLine({ title: "Mock cloud", region: "" }, [], titleOf)).toBe("Mock cloud");
  });

  it("names every cloud a split deployment builds on, but not a placement with no block", () => {
    const empty: Placement = { ...lab, connection_id: "c-empty", node_ids: [] };
    expect(cloudsLine({ title: "Amazon Web Services", region: "ca-central-1" }, [lab, empty], titleOf)).toBe(
      "Amazon Web Services (ca-central-1) and OpenStack (RegionOne)",
    );
  });
});

describe("movePlacement", () => {
  it("places a block on a new cloud, with empty settings", () => {
    expect(movePlacement("c-aws", [], blocks, "files", "c-lab")).toEqual([{ connection_id: "c-lab", node_ids: ["files"], settings: {} }]);
  });

  it("adds a block to a cloud that already has one, keeping its settings", () => {
    expect(movePlacement("c-aws", [lab], blocks, "web", "c-lab")).toEqual([
      { connection_id: "c-lab", node_ids: ["db", "web"], settings: { image: "cirros" } },
    ]);
  });

  it("moves a block between clouds, and drops a cloud left with none", () => {
    expect(movePlacement("c-aws", [lab], blocks, "db", "c-other")).toEqual([{ connection_id: "c-other", node_ids: ["db"], settings: {} }]);
  });

  it("gives a block back to the deployment's own cloud", () => {
    expect(movePlacement("c-aws", [lab], blocks, "db", "c-aws")).toEqual([]);
  });

  it("forgets blocks no longer in the diagram", () => {
    const stale: Placement = { ...lab, node_ids: ["db", "gone"] };
    expect(movePlacement("c-aws", [stale], blocks, "web", "c-lab")).toEqual([
      { connection_id: "c-lab", node_ids: ["db", "web"], settings: { image: "cirros" } },
    ]);
    expect(movePlacement("c-aws", [], blocks, "gone", "c-lab")).toEqual([]);
  });
});

describe("withPlacementSettings", () => {
  it("replaces one cloud's settings only", () => {
    expect(withPlacementSettings([lab], "c-lab", { image: "ubuntu" })).toEqual([{ connection_id: "c-lab", node_ids: ["db"], settings: { image: "ubuntu" } }]);
  });
});

describe("cloudChoices", () => {
  const primary = conn("c-aws", "aws");
  const offered = (c: string) => c !== "gone-cloud";
  it("lists the deployment's own cloud first, then other working ones this server offers", () => {
    const all = [primary, conn("c-lab", "general-openstack"), conn("c-broken", "general-openstack", "failed"), conn("c-gone", "gone-cloud")];
    expect(cloudChoices(primary, all, offered, new Set()).map((c) => c.id)).toEqual(["c-aws", "c-lab"]);
  });
  it("keeps a failed connection a block already sits on", () => {
    const all = [primary, conn("c-broken", "general-openstack", "failed")];
    expect(cloudChoices(primary, all, offered, new Set(["c-broken"])).map((c) => c.id)).toEqual(["c-aws", "c-broken"]);
  });
});

describe("placementReason", () => {
  it("lets blocks move only while the deployment holds nothing", () => {
    expect(placementReason("new", false)).toBeNull();
    expect(placementReason("destroyed", false)).toBeNull();
    expect(placementReason("live", false)).toMatch(/Destroy it first/);
    expect(placementReason("failed", false)).toMatch(/Destroy it first/);
    expect(placementReason("new", true)).toMatch(/run is in progress/);
  });
});

const split: Compatibility = {
  ready: false,
  problems: [
    { code: "cross_cloud_edge", edge_source: "web", edge_target: "net", message: "web and net are wired together inside one cloud." },
    { code: "unsupported_block", node_id: "files", message: "OpenStack cannot build a Bucket block." },
  ],
  notes: [
    { code: "cross_cloud_key", edge_source: "web", edge_target: "files", message: "Clutter creates a key that can only read files and puts it on web." },
    { code: "cross_cloud_address", edge_source: "web", edge_target: "db", message: "web (aws) gets the public address of db (lab) when it first starts." },
  ],
  clouds: [
    { connection_id: "c-aws", connection_name: "aws", connector: "aws", position: 1, ready: true, problems: [], notes: [] },
    {
      connection_id: "c-lab",
      connection_name: "lab",
      connector: "general-openstack",
      position: 0,
      ready: false,
      problems: [{ code: "unsupported_block", node_id: "files", message: "OpenStack cannot build a Bucket block." }],
      notes: [],
    },
  ],
};

describe("split reports", () => {
  it("words what crosses clouds, keys first", () => {
    expect(crossNotes(split)).toEqual({
      keys: ["Clutter creates a key that can only read files and puts it on web."],
      addresses: ["web (aws) gets the public address of db (lab) when it first starts."],
    });
  });
  it("keeps only what no cloud's own report says", () => {
    const only = splitOnly(split);
    expect(only.problems.map((p) => p.code)).toEqual(["cross_cloud_edge"]);
    expect(only.notes).toEqual([]);
  });
  it("orders the clouds as a run takes them", () => {
    expect(orderedClouds(split).map((c) => c.connection_id)).toEqual(["c-lab", "c-aws"]);
    expect(orderedClouds({ ready: true, problems: [], notes: [] })).toEqual([]);
  });
});

describe("partLines", () => {
  it("is one line per cloud of a split run", () => {
    expect(
      partLines([
        { connection_id: "c-lab", name: "lab", step: "rollback", status: "rolled_back" },
        { connection_id: "c-aws", name: "aws", step: "apply", status: "failed" },
        { connection_id: "c-x", name: "x", step: "apply", status: "skipped" },
        { connection_id: "c-y", name: "y", step: "plan", status: "queued" },
      ]).map((l) => l.text),
    ).toEqual(["lab: rolled back", "aws: deploy failed", "x: deploy not started, because an earlier cloud failed", "y: plan waiting to start"]);
    expect(partLines(null)).toEqual([]);
  });
});

describe("placeBlock", () => {
  it("moves a block to a cloud and puts it back in the deployment, so it is built somewhere", () => {
    expect(placeBlock("c-os", [], ["files", "gone"], blocks, "files", "c-aws")).toEqual({
      placements: [{ connection_id: "c-aws", node_ids: ["files"], settings: {} }],
      exclusions: [],
    });
  });

  it("keeps the other left-out blocks", () => {
    expect(placeBlock("c-aws", [lab], ["web", "files"], blocks, "db", "c-aws").exclusions).toEqual(["web", "files"]);
  });
});

describe("cloudBlocks", () => {
  it("lists the blocks one connection of a deployment runs", () => {
    expect(cloudBlocks("c-aws", [lab], blocks, "c-lab").map((b) => b.id)).toEqual(["db"]);
    expect(cloudBlocks("c-aws", [lab], blocks, "c-aws").map((b) => b.id)).toEqual(["web", "files"]);
  });
});

describe("deploymentTitle", () => {
  it("names every connection a split deployment builds on", () => {
    expect(deploymentTitle("devstack", [{ ...lab, connection_name: "Local AWS" }])).toBe("devstack + Local AWS");
    expect(deploymentTitle("devstack", [{ ...lab, node_ids: [] }])).toBe("devstack");
    expect(deploymentTitle("devstack", null)).toBe("devstack");
  });
});

describe("needsTwoClouds", () => {
  it("is true when blocks are placed elsewhere or the own cloud cannot build some", () => {
    expect(needsTwoClouds([lab], 0)).toBe(true);
    expect(needsTwoClouds([], 1)).toBe(true);
    expect(needsTwoClouds([{ ...lab, node_ids: [] }], 0)).toBe(false);
    expect(needsTwoClouds(null, 0)).toBe(false);
  });
});

describe("placementSuggestions", () => {
  const offers = (connector: string, kind: string) => connector === "aws" || kind !== "bucket";
  const files = blocks[2];

  it("suggests the first other working connection whose cloud can build the block", () => {
    const connections = [conn("c-os", "general-openstack"), conn("c-broken", "aws", "failed"), conn("c-aws", "aws")];
    expect(placementSuggestions([files], "c-os", connections, offers)).toEqual([{ block: files, connection: connections[2] }]);
  });

  it("suggests nothing when no other connection can build it", () => {
    expect(placementSuggestions([files], "c-os", [conn("c-os", "general-openstack"), conn("c-os2", "general-openstack")], offers)).toEqual([]);
  });

  it("groups the chosen blocks into one placement per connection", () => {
    expect(
      suggestedPlacements([
        { blockId: "files", connectionId: "c-aws" },
        { blockId: "db", connectionId: "c-aws" },
        { blockId: "web", connectionId: "c-lab" },
      ]),
    ).toEqual([
      { connection_id: "c-aws", node_ids: ["db", "files"], settings: {} },
      { connection_id: "c-lab", node_ids: ["web"], settings: {} },
    ]);
  });
});
