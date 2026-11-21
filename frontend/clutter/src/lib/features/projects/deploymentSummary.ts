// What a project's deployments add up to, for the Projects page and a
// project's Deployments tab and settings. The rows are B's organization
// deployment list (deployments/orgApi.ts). A deployment's named state and its
// label are runState.ts's (deploymentState, DEPLOYMENT_STATE).

// Relative imports: Vitest runs this file without the "@/" alias.
import { DEPLOYMENT_STATE, deploymentState } from "../deployments/runState";
import type { DeploymentStatus, Run } from "../deployments/types";

type Row = { project_id: string; status: DeploymentStatus; latest_run: Run | null };

// A deployment that is neither new nor destroyed may still hold resources in
// the cloud, so B refuses to delete what it belongs to (409).
export const holdsResources = (status: DeploymentStatus) => status !== "new" && status !== "destroyed";

export type DeploySummary = {
  // Built and working (an update in progress still counts).
  live: number;
  // A run is queued or running.
  running: number;
  // May hold resources (live or failed).
  holding: number;
};

const empty = (): DeploySummary => ({ live: 0, running: 0, holding: 0 });

function add(s: DeploySummary, row: Row) {
  if (row.status === "live") s.live++;
  if (DEPLOYMENT_STATE[deploymentState(row)].running) s.running++;
  if (holdsResources(row.status)) s.holding++;
}

export function summarize(rows: readonly Row[]): DeploySummary {
  const s = empty();
  for (const row of rows) add(s, row);
  return s;
}

// Whether a project or team may be deleted, from the check of its
// deployments: "checking" while the list loads, "failed" when it could not be
// read (delete stays off, since A refuses a delete while a deployment holds
// resources and the page cannot tell), "holding" with the count that may hold
// resources, else "clear".
export type DeleteCheck = { state: "checking" | "failed" | "holding" | "clear"; holding: number };

export function deleteCheck(q: { isError: boolean; data?: readonly Row[] | null }): DeleteCheck {
  if (q.data) {
    const holding = summarize(q.data).holding;
    return { state: holding > 0 ? "holding" : "clear", holding };
  }
  return { state: q.isError ? "failed" : "checking", holding: 0 };
}

// summarizeByProject keys each project's summary by its id. A project with no
// deployment has no entry.
export function summarizeByProject(rows: readonly Row[]): Map<string, DeploySummary> {
  const out = new Map<string, DeploySummary>();
  for (const row of rows) {
    const s = out.get(row.project_id) ?? empty();
    add(s, row);
    out.set(row.project_id, s);
  }
  return out;
}

// lastRunAt is when the deployment's latest run finished, or started, or was
// asked for; null when it has never run.
export function lastRunAt(row: { latest_run: Run | null }): string | null {
  const run = row.latest_run;
  return run ? (run.finished_at ?? run.started_at ?? run.created_at) : null;
}
