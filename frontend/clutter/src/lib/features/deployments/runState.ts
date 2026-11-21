// Where a deployment stands, in one named state, and the one primary action
// that state offers. The deploy panel, the project's Deployments tab and the
// editor's live markers read these, so every screen names a state the same way.

// Relative imports: Vitest runs this file without the "@/" alias.
import { changesNotDeployed, isActive, type DeploymentListItem, type Run } from "./types";

// "destroyed" is a destroy that finished: the deployment stays on the diagram
// (it says "Destroyed", never "Removed", which is the word for taking it off
// the diagram).
export type DeploymentState = "not_deployed" | "queued" | "planning" | "applying" | "destroying" | "live" | "failed" | "destroyed";

export type StateTone = "neutral" | "primary" | "success" | "warning" | "danger" | "info";

export const DEPLOYMENT_STATE: Record<DeploymentState, { label: string; tone: StateTone; running: boolean; hint: string }> = {
  not_deployed: { label: "Not deployed", tone: "neutral", running: false, hint: "Nothing is built in your cloud yet." },
  queued: { label: "Queued", tone: "info", running: true, hint: "The run is waiting to start." },
  planning: { label: "Planning", tone: "info", running: true, hint: "Clutter is working out what would change. Nothing is built." },
  applying: { label: "Applying", tone: "primary", running: true, hint: "Clutter is building it in your cloud." },
  destroying: { label: "Destroying", tone: "warning", running: true, hint: "Clutter is removing what it built." },
  live: { label: "Live", tone: "success", running: false, hint: "Built in your cloud." },
  failed: { label: "Failed", tone: "danger", running: false, hint: "The last run failed. Read the log, fix the cause, then try again." },
  destroyed: { label: "Destroyed", tone: "neutral", running: false, hint: "Everything it built is gone from your cloud." },
};

// deploymentState names where a deployment stands: a run in progress first,
// then what the deployment holds.
export function deploymentState(item: Pick<DeploymentListItem, "status" | "latest_run">): DeploymentState {
  const run = item.latest_run;
  if (run && isActive(run)) {
    if (run.status === "queued") return "queued";
    return run.action === "plan" ? "planning" : run.action === "destroy" ? "destroying" : "applying";
  }
  switch (item.status) {
    case "live":
      return "live";
    case "failed":
      return "failed";
    case "destroyed":
      return "destroyed";
    default:
      return "not_deployed";
  }
}

// The one primary action a state offers. "deploy" builds it; "update"
// builds the saved changes; "up_to_date" is the same button, off, because the
// cloud already holds the saved diagram; "retry" deploys again after a failed
// deploy; "retry_destroy" destroys again after a failed destroy, so a user
// who asked to remove everything is never offered to build it; "log" follows
// a run in progress.
export type PrimaryAction = { kind: "deploy" | "update" | "up_to_date" | "retry" | "retry_destroy" | "log"; label: string };

export function primaryAction(item: Pick<DeploymentListItem, "status" | "latest_run" | "last_apply" | "diagram_version">): PrimaryAction {
  const state = deploymentState(item);
  if (DEPLOYMENT_STATE[state].running) return { kind: "log", label: "Show log" };
  if (state === "failed") {
    return item.latest_run?.action === "destroy" ? { kind: "retry_destroy", label: "Destroy again" } : { kind: "retry", label: "Deploy again" };
  }
  if (state === "live") return changesNotDeployed(item) ? { kind: "update", label: "Update" } : { kind: "up_to_date", label: "Up to date" };
  return { kind: "deploy", label: "Deploy" };
}

// primaryRun is the run the primary button asks to start (behind its
// confirmation): a destroy for "retry_destroy", an apply for the building
// kinds, none for "log" and "up_to_date".
export function primaryRun(kind: PrimaryAction["kind"]): "apply" | "destroy" | null {
  if (kind === "retry_destroy") return "destroy";
  if (kind === "log" || kind === "up_to_date") return null;
  return "apply";
}

// A run's line in the history: "Deploy", "Plan", "Destroy" and how it ended.
export const RUN_ACTION: Record<Run["action"], string> = { plan: "Plan", apply: "Deploy", destroy: "Destroy" };
const RUN_END: Record<Run["status"], string> = { queued: "queued", running: "running", completed: "finished", failed: "failed" };

export const runLabel = (run: Pick<Run, "action" | "status">) => `${RUN_ACTION[run.action]} ${RUN_END[run.status]}`;

// runEndToast is what replaces a run's "started" toast once the run ends
// ("Deploy finished on lab", "Destroy failed on lab"), or null while it is
// still queued or running.
export function runEndToast(run: Pick<Run, "action" | "status">, title: string): { ok: boolean; text: string } | null {
  if (run.status !== "completed" && run.status !== "failed") return null;
  return { ok: run.status === "completed", text: `${runLabel(run)} on ${title}` };
}

// mergeRuns adds the runs B returned to the ones already known for a
// deployment: newest first, each run once (the latest copy wins), at most
// limit of them. B returns only a deployment's latest run and its last
// successful apply, so the history holds those and the runs seen since the
// page opened.
export function mergeRuns(known: Run[], incoming: (Run | null | undefined)[], limit = 20): Run[] {
  const byId = new Map(known.map((r) => [r.id, r]));
  for (const r of incoming) if (r) byId.set(r.id, r);
  return [...byId.values()].sort((a, b) => (a.created_at < b.created_at ? 1 : a.created_at > b.created_at ? -1 : 0)).slice(0, limit);
}
