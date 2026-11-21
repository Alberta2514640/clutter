import { describe, expect, it } from "vitest";
import { costLines, deployLine, destroyLine } from "./confirm";
import type { Placement } from "./types";

const TITLES: Record<string, string> = { vm: "VM", network: "Network", firewall: "Firewall", bucket: "Bucket" };
const titleOf = (k: string) => TITLES[k] ?? k;

const blocks = [
  { id: "app-net", kind: "network", name: "app-net" },
  { id: "app-fw", kind: "firewall", name: "app-fw" },
  { id: "app", kind: "vm", name: "app" },
  { id: "files", kind: "bucket", name: "files" },
];

const aws: Placement = { connection_id: "c-aws", connection_name: "Local AWS", connector: "aws", region: "r", node_ids: ["files"], settings: null };

describe("deployLine", () => {
  it("says what one cloud builds", () => {
    expect(deployLine({ id: "c-os", name: "lab" }, null, blocks.slice(0, 3), [], titleOf)).toBe("Builds 1 server, 1 network and 1 firewall on lab.");
  });

  it("says what each cloud of a split deployment builds", () => {
    expect(deployLine({ id: "c-os", name: "devstack" }, [aws], blocks, [], titleOf)).toBe(
      "Builds 1 server, 1 network and 1 firewall on devstack, and 1 bucket on Local AWS.",
    );
  });

  it("leaves out the left-out blocks, and a cloud left with nothing", () => {
    expect(deployLine({ id: "c-os", name: "devstack" }, [aws], blocks, ["files"], titleOf)).toBe("Builds 1 server, 1 network and 1 firewall on devstack.");
    expect(deployLine({ id: "c-os", name: "devstack" }, null, blocks, blocks.map((b) => b.id), titleOf)).toBe("Nothing to build.");
  });
});

describe("destroyLine", () => {
  it("lists what each cloud destroys, as the deploy confirmation lists what it builds", () => {
    expect(destroyLine({ id: "c-os", name: "devstack" }, [aws], blocks, [], titleOf)).toBe(
      "Destroys 1 server, 1 network and 1 firewall on devstack, and 1 bucket on Local AWS.",
    );
    expect(destroyLine({ id: "c-os", name: "lab" }, null, blocks, ["files"], titleOf)).toBe("Destroys 1 server, 1 network and 1 firewall on lab.");
    expect(destroyLine({ id: "c-os", name: "lab" }, null, [], [], titleOf)).toBe("Nothing is left to destroy.");
  });

  it("counts what the last deploy built, not blocks added since", () => {
    const log = ["Apply complete: 2 blocks built or changed.", "Built: app vm app", "Built: app-net network app-net"].join("\n");
    const now = [...blocks, { id: "disk", kind: "volume", name: "disk" }];
    expect(destroyLine({ id: "c-os", name: "lab" }, null, now, [], titleOf, log)).toBe("Destroys 1 server and 1 network on lab.");
    // A block left out after that deploy is still in the cloud.
    expect(destroyLine({ id: "c-os", name: "lab" }, null, now, ["app"], titleOf, log)).toBe("Destroys 1 server and 1 network on lab.");
  });

  it("falls back to the saved diagram when the last deploy named nothing", () => {
    for (const log of [null, undefined, "", "Apply complete! Resources: 3 added, 0 changed, 0 destroyed."]) {
      expect(destroyLine({ id: "c-os", name: "lab" }, null, blocks, ["files"], titleOf, log)).toBe("Destroys 1 server, 1 network and 1 firewall on lab.");
    }
  });
});

describe("costLines", () => {
  it("lists each cost note once, and nothing else", () => {
    const nat = "Internet access for private servers adds a NAT gateway on AWS: about $40 a month plus data.";
    const report = {
      ready: true,
      problems: null,
      notes: [
        { node_id: "a", code: "cost", message: nat },
        { node_id: "b", code: "cost", message: nat },
        { code: "edge_dropped", message: "dropped" },
      ],
    };
    expect(costLines(report)).toEqual([nat]);
    expect(costLines(null)).toEqual([]);
  });
});
