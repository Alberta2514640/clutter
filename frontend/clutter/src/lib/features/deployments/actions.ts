// What a deployment's buttons can do right now, and the plain reason when one
// cannot; and which connections "Add deployment" can offer. B refuses the same
// cases; this only explains them before the user clicks.

// Relative imports: Vitest runs this file without the "@/" alias.
import type { Connection } from "../connections/types";
import { joinAnd } from "./preview";
import type { DeploymentStatus } from "./types";

export type RowAction = "preview" | "plan" | "apply" | "destroy" | "remove";

export const ACTION_LABEL: Record<RowAction, string> = {
  preview: "Preview Terraform",
  plan: "Plan",
  apply: "Deploy",
  destroy: "Destroy",
  remove: "Remove",
};

export type RowState = {
  status: DeploymentStatus;
  // A run is queued or running, or one is being started.
  running: boolean;
  // A change to the deployment's settings or left-out blocks is saving.
  saving: boolean;
  // Why the canvas holds back checks and runs (unsaved changes), or null.
  unsaved: string | null;
  // The compatibility check has not answered yet, or failed.
  checking: boolean;
  checkFailed: boolean;
  // The cloud can build the saved diagram, minus the left-out blocks.
  ready: boolean;
};

export const REASON = {
  running: "A run is in progress. Wait for it to finish.",
  saving: "A change to this deployment is saving.",
  checking: "Clutter is still checking what this cloud can build.",
  checkFailed: "Clutter could not check what this cloud can build. Reload the page to try again.",
  notReady: "This cloud cannot build the diagram as it is. Fix the problems above, or leave those blocks out.",
  nothingBuilt: "Nothing is built yet.",
  destroyed: "Everything this deployment built is already removed.",
  live: "Destroy it first. It may still have resources in your cloud.",
} as const;

// Removing a deployment is offered only when it holds nothing in the cloud.
export const canOfferRemove = (status: DeploymentStatus) => status === "new" || status === "destroyed";

// The reason each action is off, or null when it can be used. The first
// reason that applies wins, so the user sees the one to act on now.
export function actionReasons(s: RowState): Record<RowAction, string | null> {
  const first = (...reasons: (string | null | false)[]) => reasons.find((r): r is string => !!r) ?? null;
  const check = first(s.checking && REASON.checking, s.checkFailed && REASON.checkFailed, !s.ready && REASON.notReady);
  return {
    preview: first(s.unsaved, check),
    plan: first(s.running && REASON.running, s.saving && REASON.saving, s.unsaved, check),
    apply: first(s.running && REASON.running, s.saving && REASON.saving, s.unsaved, check),
    // A destroy does not read the diagram, so unsaved changes do not stop it.
    destroy: first(
      s.running && REASON.running,
      s.saving && REASON.saving,
      s.status === "new" && REASON.nothingBuilt,
      s.status === "destroyed" && REASON.destroyed,
    ),
    remove: first(s.running && REASON.running, s.saving && REASON.saving, !canOfferRemove(s.status) && REASON.live),
  };
}

// One line per distinct reason, naming the buttons it turns off, in the order
// the buttons are shown: "Plan and Deploy are off. A run is in progress. ..."
export function offLines(reasons: Partial<Record<RowAction, string | null>>, shown: RowAction[]): string[] {
  const byReason = new Map<string, string[]>();
  for (const a of shown) {
    const r = reasons[a];
    if (!r) continue;
    byReason.set(r, [...(byReason.get(r) ?? []), ACTION_LABEL[a]]);
  }
  return [...byReason].map(([reason, labels]) => `${joinAnd(labels)} ${labels.length === 1 ? "is" : "are"} off. ${reason}`);
}

// What "Add deployment" can offer. "none": the organization has no
// connection. "all_used": every connection already has a deployment of this
// diagram. "none_working": the free ones all failed their last check. "pick":
// at least one is free and working.
export type AddChoice = {
  state: "none" | "all_used" | "none_working" | "pick";
  usable: Connection[];
  // Free connections that failed their last check, to name on the form.
  broken: Connection[];
};

export function addDeploymentChoice(connections: Connection[], usedConnectionIds: ReadonlySet<string>): AddChoice {
  if (connections.length === 0) return { state: "none", usable: [], broken: [] };
  const free = connections.filter((c) => !usedConnectionIds.has(c.id));
  if (free.length === 0) return { state: "all_used", usable: [], broken: [] };
  const usable = free.filter((c) => c.status === "working");
  const broken = free.filter((c) => c.status !== "working");
  return { state: usable.length === 0 ? "none_working" : "pick", usable, broken };
}
