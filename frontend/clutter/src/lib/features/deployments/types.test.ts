import { describe, expect, it } from "vitest";
import { changesNotDeployed, type Run } from "./types";

const apply = (version: number | null): Run => ({
  id: "r1",
  deployment_id: "d1",
  action: "apply",
  status: "completed",
  diagram_version: version,
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
});
