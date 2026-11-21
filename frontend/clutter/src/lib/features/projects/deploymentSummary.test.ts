import { describe, expect, it } from "vitest";
import type { DeploymentStatus, Run } from "../deployments/types";
import { deleteCheck, holdsResources, lastRunAt, summarize, summarizeByProject } from "./deploymentSummary";

const run = (action: Run["action"], status: Run["status"], at: Partial<Run> = {}): Run => ({
  id: "r",
  deployment_id: "d",
  action,
  status,
  diagram_version: 1,
  failure: null,
  created_at: "2026-11-20T10:00:00Z",
  started_at: null,
  finished_at: null,
  ...at,
});

const row = (project_id: string, status: DeploymentStatus, latest_run: Run | null = null) => ({ project_id, status, latest_run });

describe("deleteCheck", () => {
  it("keeps delete off while checking and when the check failed", () => {
    expect(deleteCheck({ isError: false })).toEqual({ state: "checking", holding: 0 });
    expect(deleteCheck({ isError: true })).toEqual({ state: "failed", holding: 0 });
    expect(deleteCheck({ isError: true, data: null })).toEqual({ state: "failed", holding: 0 });
  });

  it("counts what may hold resources, and is clear when nothing does", () => {
    expect(deleteCheck({ isError: false, data: [row("p", "live"), row("p", "failed"), row("p", "destroyed")] })).toEqual({ state: "holding", holding: 2 });
    expect(deleteCheck({ isError: false, data: [row("p", "new"), row("p", "destroyed")] })).toEqual({ state: "clear", holding: 0 });
    expect(deleteCheck({ isError: false, data: [] })).toEqual({ state: "clear", holding: 0 });
  });

  it("goes by the last list read when a later check fails", () => {
    expect(deleteCheck({ isError: true, data: [row("p", "live")] })).toEqual({ state: "holding", holding: 1 });
  });
});

describe("holdsResources", () => {
  it("is true only while the cloud may still hold something", () => {
    expect(holdsResources("live")).toBe(true);
    expect(holdsResources("failed")).toBe(true);
    expect(holdsResources("new")).toBe(false);
    expect(holdsResources("destroyed")).toBe(false);
  });
});

describe("summarize", () => {
  it("counts live, running and holding deployments", () => {
    const rows = [row("p1", "live"), row("p1", "failed"), row("p2", "new", run("apply", "running")), row("p2", "destroyed")];
    expect(summarize(rows)).toEqual({ live: 1, running: 1, holding: 2 });
    const byProject = summarizeByProject(rows);
    expect(byProject.get("p1")).toEqual({ live: 1, running: 0, holding: 2 });
    expect(byProject.get("p2")).toEqual({ live: 0, running: 1, holding: 0 });
    expect(byProject.get("p3")).toBeUndefined();
  });
});

describe("lastRunAt", () => {
  it("prefers the finish, then the start, then the request", () => {
    expect(lastRunAt({ latest_run: null })).toBeNull();
    expect(lastRunAt({ latest_run: run("apply", "queued") })).toBe("2026-11-20T10:00:00Z");
    expect(lastRunAt({ latest_run: run("apply", "running", { started_at: "2026-11-20T10:01:00Z" }) })).toBe("2026-11-20T10:01:00Z");
    expect(
      lastRunAt({ latest_run: run("apply", "completed", { started_at: "2026-11-20T10:01:00Z", finished_at: "2026-11-20T10:05:00Z" }) }),
    ).toBe("2026-11-20T10:05:00Z");
  });
});
