import { describe, expect, it } from "vitest";
import { changesNotDeployed, diagramNotDeployed, type Run } from "./types";

const apply = (version: number | null, config?: number | null): Run => ({
  id: "r1",
  deployment_id: "d1",
  action: "apply",
  status: "completed",
  diagram_version: version,
  config_version: config,
  failure: null,
  created_at: "",
  started_at: null,
  finished_at: null,
});

describe("changesNotDeployed", () => {
  it("is true only when the saved diagram is newer than the one the last apply built, and it is still built", () => {
    expect(changesNotDeployed({ status: "live", last_apply: apply(3), diagram_version: 4 })).toBe(true);
    expect(changesNotDeployed({ status: "failed", last_apply: apply(3), diagram_version: 4 })).toBe(true);
    expect(changesNotDeployed({ status: "live", last_apply: apply(4), diagram_version: 4 })).toBe(false);
    expect(changesNotDeployed({ status: "destroyed", last_apply: apply(3), diagram_version: 4 })).toBe(false);
    expect(changesNotDeployed({ status: "live", last_apply: null, diagram_version: 4 })).toBe(false);
    expect(changesNotDeployed({ status: "live", last_apply: apply(null), diagram_version: 4 })).toBe(false);
  });

  it("is true when the deployment's settings changed since the last apply, with the diagram unchanged", () => {
    expect(changesNotDeployed({ status: "live", last_apply: apply(4, 2), diagram_version: 4, config_version: 3 })).toBe(true);
    expect(changesNotDeployed({ status: "failed", last_apply: apply(4, 2), diagram_version: 4, config_version: 3 })).toBe(true);
    expect(changesNotDeployed({ status: "live", last_apply: apply(4, 3), diagram_version: 4, config_version: 3 })).toBe(false);
    expect(changesNotDeployed({ status: "destroyed", last_apply: apply(4, 2), diagram_version: 4, config_version: 3 })).toBe(false);
    expect(changesNotDeployed({ status: "new", last_apply: null, diagram_version: 4, config_version: 3 })).toBe(false);
  });

  it("counts an apply from before config_version as 0, so a deployment that never changed is up to date", () => {
    // A live deployment from before the column: both at the start.
    expect(changesNotDeployed({ status: "live", last_apply: apply(4, null), diagram_version: 4, config_version: 0 })).toBe(false);
    expect(changesNotDeployed({ status: "live", last_apply: apply(4), diagram_version: 4, config_version: 0 })).toBe(false);
    // Its first settings change since then shows.
    expect(changesNotDeployed({ status: "live", last_apply: apply(4, null), diagram_version: 4, config_version: 1 })).toBe(true);
    // A B that sends no config_version compares only the diagram.
    expect(changesNotDeployed({ status: "live", last_apply: apply(4, 2), diagram_version: 4 })).toBe(false);
    expect(changesNotDeployed({ status: "live", last_apply: apply(4, 2), diagram_version: 4, config_version: null })).toBe(false);
  });
});

describe("diagramNotDeployed", () => {
  it("compares only the diagram, not the deployment's settings", () => {
    expect(diagramNotDeployed({ status: "live", last_apply: apply(3, 5), diagram_version: 4 })).toBe(true);
    expect(diagramNotDeployed({ status: "live", last_apply: apply(4, 2), diagram_version: 4 })).toBe(false);
    expect(diagramNotDeployed({ status: "new", last_apply: apply(3), diagram_version: 4 })).toBe(false);
  });
});
