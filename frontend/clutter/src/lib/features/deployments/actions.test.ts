import { describe, expect, it } from "vitest";
import type { Connection } from "../connections/types";
import { actionReasons, addDeploymentChoice, allLeftOut, REASON, startingConnection, type RowState } from "./actions";

const base: RowState = { status: "new", running: false, saving: false, unsaved: null, checking: false, checkFailed: false, ready: true, allLeftOut: false, noBlocks: false };
const unsaved = "Save the diagram first. Deployments use the saved version.";

describe("actionReasons", () => {
  it("lets a ready, new deployment preview, plan and deploy, but not destroy", () => {
    const r = actionReasons(base);
    expect(r.preview).toBeNull();
    expect(r.plan).toBeNull();
    expect(r.apply).toBeNull();
    expect(r.destroy).toBe(REASON.nothingBuilt);
    expect(r.remove).toBeNull();
  });

  it("names the run in progress before any other reason", () => {
    const r = actionReasons({ ...base, status: "live", running: true, unsaved, ready: false });
    expect(r.plan).toBe(REASON.running);
    expect(r.apply).toBe(REASON.running);
    expect(r.destroy).toBe(REASON.running);
    expect(r.preview).toBe(unsaved);
  });

  it("holds back plan, deploy and preview for unsaved changes, but not destroy", () => {
    const r = actionReasons({ ...base, status: "live", unsaved });
    expect(r.preview).toBe(unsaved);
    expect(r.plan).toBe(unsaved);
    expect(r.destroy).toBeNull();
    expect(r.remove).toBe(REASON.live);
  });

  it("explains a cloud that cannot build the diagram, and a check still running", () => {
    expect(actionReasons({ ...base, ready: false }).apply).toBe(REASON.notReady);
    expect(actionReasons({ ...base, ready: false, checking: true }).apply).toBe(REASON.checking);
    expect(actionReasons({ ...base, ready: false, checkFailed: true }).preview).toBe(REASON.checkFailed);
  });

  it("turns plan and deploy off when every block is left out", () => {
    const r = actionReasons({ ...base, allLeftOut: true });
    expect(r.plan).toBe("Every block is left out of this deployment.");
    expect(r.apply).toBe(REASON.allLeftOut);
    expect(r.preview).toBeNull();
    expect(actionReasons({ ...base, status: "live", allLeftOut: true }).destroy).toBeNull();
    expect(allLeftOut(["a", "b"], ["b", "a", "gone"])).toBe(true);
    expect(allLeftOut(["a", "b"], ["a"])).toBe(false);
    expect(allLeftOut([], [])).toBe(false);
    expect(actionReasons({ ...base, noBlocks: true }).apply).toBe(REASON.noBlocks);
    expect(actionReasons({ ...base, noBlocks: true }).plan).toBe(REASON.noBlocks);
  });

  it("offers remove after a destroy", () => {
    const r = actionReasons({ ...base, status: "destroyed" });
    expect(r.remove).toBeNull();
    expect(r.destroy).toBe(REASON.destroyed);
  });
});

const conn = (id: string, status: Connection["status"] = "working"): Connection => ({
  id,
  organization_id: "o",
  connector: "mock",
  name: id,
  region: "r",
  status,
  status_detail: null,
  created_at: "",
  updated_at: "",
  verified_at: null,
});

describe("addDeploymentChoice", () => {
  it("separates no connections, all used, none working, and a choice", () => {
    expect(addDeploymentChoice([], new Set()).state).toBe("none");
    expect(addDeploymentChoice([conn("a")], new Set(["a"])).state).toBe("all_used");
    const broken = addDeploymentChoice([conn("a"), conn("b", "failed")], new Set(["a"]));
    expect(broken.state).toBe("none_working");
    expect(broken.broken.map((c) => c.id)).toEqual(["b"]);
    const pick = addDeploymentChoice([conn("a"), conn("b", "failed"), conn("c")], new Set());
    expect(pick.state).toBe("pick");
    expect(pick.usable.map((c) => c.id)).toEqual(["a", "c"]);
    expect(pick.broken.map((c) => c.id)).toEqual(["b"]);
  });
});

describe("startingConnection", () => {
  it("picks the only free connection, and none among several", () => {
    expect(startingConnection([conn("a")])).toBe("a");
    expect(startingConnection([conn("a"), conn("c")])).toBe("");
    expect(startingConnection([])).toBe("");
  });

  it("prefers a connection of a split example's first cloud", () => {
    const os = { ...conn("o"), connector: "general-openstack" };
    expect(startingConnection([conn("a"), os], "general-openstack")).toBe("o");
    expect(startingConnection([conn("a")], "general-openstack")).toBe("a");
  });
});
