import { describe, expect, it } from "vitest";
import type { Run } from "../deployments/types";
import { liveMarkers } from "./live";

const apply = (diagram_version: number): Run => ({
  id: "r1",
  deployment_id: "d1",
  action: "apply",
  status: "completed",
  diagram_version,
  failure: null,
  created_at: "",
  started_at: null,
  finished_at: null,
});

const deployment = {
  status: "live" as const,
  last_apply: apply(1),
  diagram_version: 1,
  exclusions: [] as string[],
  placements: null,
  connection_name: "lab",
};

const log = ["Planning...", "Apply complete: 2 blocks built or changed.", "Built: web vm web", "Built: net network net"].join("\n");

describe("liveMarkers", () => {
  it("marks Live only the blocks the last apply built, and a block drawn since as not built", () => {
    const m = liveMarkers({ ...deployment, diagram_version: 2 }, log, ["web", "net"], ["web", "net", "db"]);
    expect([...m.live]).toEqual([
      ["web", "Live on lab"],
      ["net", "Live on lab"],
    ]);
    expect([...m.notBuilt]).toEqual(["db"]);
  });

  it("names the connection a placed block runs on, and leaves out what the deployment leaves out", () => {
    const placed = {
      ...deployment,
      exclusions: ["db"],
      placements: [{ connection_id: "c2", connection_name: "edge", connector: "aws", region: "", node_ids: ["net"], settings: null }],
    };
    const m = liveMarkers(placed, log, [], ["web", "net", "db"]);
    expect(m.live.get("net")).toBe("Live on edge");
    expect(m.notBuilt.has("db")).toBe(false);
  });

  it("marks nothing while the log loads, with no apply, or once destroyed", () => {
    expect(liveMarkers(deployment, null, ["web"], ["web"]).live.size).toBe(0);
    expect(liveMarkers({ ...deployment, last_apply: null }, log, ["web"], ["web"]).live.size).toBe(0);
    expect(liveMarkers({ ...deployment, status: "destroyed" }, log, ["web"], ["web"]).live.size).toBe(0);
  });

  it("lets the saved blocks stand in for a log without Built lines, only while the diagram is unchanged", () => {
    const old = "Apply complete: 1 block built or changed.";
    expect([...liveMarkers(deployment, old, ["web"], ["web", "db"]).live.keys()]).toEqual(["web"]);
    const changed = liveMarkers({ ...deployment, diagram_version: 2 }, old, ["web"], ["web", "db"]);
    expect(changed.live.size).toBe(0);
    expect(changed.notBuilt.size).toBe(0);
    // A change to the deployment's left-out blocks or settings since the
    // apply counts too: the saved blocks may not be what it built.
    expect(liveMarkers({ ...deployment, config_version: 1 }, old, ["web"], ["web", "db"]).live.size).toBe(0);
  });
});
