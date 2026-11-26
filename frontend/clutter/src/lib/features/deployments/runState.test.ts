import { describe, expect, it } from "vitest";
import { DEPLOYMENT_STATE, deploymentState, mergeRuns, primaryAction, primaryRun, runEndToast, runLabel } from "./runState";
import type { Run } from "./types";

const run = (id: string, over: Partial<Run> = {}): Run => ({
  id,
  deployment_id: "d1",
  action: "apply",
  status: "completed",
  diagram_version: 1,
  failure: null,
  created_at: `2026-11-20T10:00:0${id}Z`,
  started_at: null,
  finished_at: null,
  ...over,
});

describe("deploymentState", () => {
  it("names a run in progress by what it does", () => {
    expect(deploymentState({ status: "new", latest_run: run("1", { status: "queued" }) })).toBe("queued");
    expect(deploymentState({ status: "new", latest_run: run("1", { status: "running", action: "plan" }) })).toBe("planning");
    expect(deploymentState({ status: "new", latest_run: run("1", { status: "running" }) })).toBe("applying");
    expect(deploymentState({ status: "live", latest_run: run("1", { status: "running", action: "destroy" }) })).toBe("destroying");
  });

  it("names what the deployment holds when nothing runs", () => {
    expect(deploymentState({ status: "new", latest_run: null })).toBe("not_deployed");
    expect(deploymentState({ status: "live", latest_run: run("1") })).toBe("live");
    expect(deploymentState({ status: "failed", latest_run: run("1", { status: "failed" }) })).toBe("failed");
    expect(deploymentState({ status: "destroyed", latest_run: run("1", { action: "destroy" }) })).toBe("destroyed");
  });

  it("says Destroyed after a destroy, never Removed, the word for taking it off the diagram", () => {
    expect(DEPLOYMENT_STATE.destroyed.label).toBe("Destroyed");
    expect(Object.values(DEPLOYMENT_STATE).some((s) => s.label === "Removed")).toBe(false);
  });
});

describe("runEndToast", () => {
  it("replaces the started toast once the run ends, and not before", () => {
    expect(runEndToast(run("1", { status: "completed" }), "lab")).toEqual({ ok: true, text: "Deploy finished on lab" });
    expect(runEndToast(run("1", { status: "failed", action: "destroy" }), "lab")).toEqual({ ok: false, text: "Destroy failed on lab" });
    expect(runEndToast(run("1", { status: "running" }), "lab")).toBeNull();
    expect(runEndToast(run("1", { status: "queued" }), "lab")).toBeNull();
  });
});

describe("primaryAction", () => {
  it("offers Deploy before anything is built and after a destroy", () => {
    expect(primaryAction({ status: "new", latest_run: null }).label).toBe("Deploy");
    expect(primaryAction({ status: "destroyed", latest_run: run("1", { action: "destroy" }) }).label).toBe("Deploy");
  });

  it("offers Update only when the saved diagram is newer than the last deploy", () => {
    const built = run("1", { diagram_version: 2 });
    expect(primaryAction({ status: "live", latest_run: built, last_apply: built, diagram_version: 3 })).toEqual({ kind: "update", label: "Update" });
    expect(primaryAction({ status: "live", latest_run: built, last_apply: built, diagram_version: 2 })).toEqual({
      kind: "up_to_date",
      label: "Up to date",
    });
  });

  it("follows a run in progress and retries a failure", () => {
    expect(primaryAction({ status: "new", latest_run: run("1", { status: "running" }) }).kind).toBe("log");
    expect(primaryAction({ status: "failed", latest_run: run("1", { status: "failed" }) }).label).toBe("Deploy again");
  });

  it("offers Destroy again, never Deploy again, after a failed destroy", () => {
    expect(primaryAction({ status: "failed", latest_run: run("1", { status: "failed", action: "destroy" }) })).toEqual({
      kind: "retry_destroy",
      label: "Destroy again",
    });
  });

  it("starts a destroy from Destroy again and an apply from the building actions", () => {
    expect(primaryRun("retry_destroy")).toBe("destroy");
    expect(primaryRun("retry")).toBe("apply");
    expect(primaryRun("deploy")).toBe("apply");
    expect(primaryRun("update")).toBe("apply");
    expect(primaryRun("up_to_date")).toBeNull();
    expect(primaryRun("log")).toBeNull();
  });
});

describe("runLabel", () => {
  it("says the action and how it ended", () => {
    expect(runLabel({ action: "apply", status: "completed" })).toBe("Deploy finished");
    expect(runLabel({ action: "destroy", status: "failed" })).toBe("Destroy failed");
  });
});

describe("mergeRuns", () => {
  it("keeps each run once, newest first, with the latest copy", () => {
    const first = run("1", { status: "running" });
    const known = mergeRuns([], [first]);
    const merged = mergeRuns(known, [run("1"), run("2", { action: "destroy" }), null]);
    expect(merged.map((r) => r.id)).toEqual(["2", "1"]);
    expect(merged[1].status).toBe("completed");
  });

  it("caps the history", () => {
    expect(mergeRuns([], [run("1"), run("2"), run("3")], 2).map((r) => r.id)).toEqual(["3", "2"]);
  });
});
