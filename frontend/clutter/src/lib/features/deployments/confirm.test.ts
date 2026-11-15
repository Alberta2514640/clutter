import { describe, expect, it } from "vitest";
import { costLines, deployLine } from "./confirm";
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
