// Mirrors backend/adapter/internal/api/handlers.go (deploymentView,
// deploymentListItem, runView), backend/adapter/internal/api/split.go
// (placementView, compatibilityView, cloudReport), backend/adapter/internal/jobs
// (PartProgress), backend/adapter/internal/capability (Report) and
// connectors.Problem.

export type DeploymentStatus = "new" | "live" | "failed" | "destroyed";

export type RunAction = "plan" | "apply" | "destroy";

export type RunStatus = "queued" | "running" | "completed" | "failed";

// Where one cloud of a split run stands. "rollback" is a step that destroys
// what the run built after a later cloud failed.
export type PartStep = RunAction | "rollback";

export type PartStatus = RunStatus | "skipped" | "rolled_back";

export type PartProgress = {
  connection_id: string;
  name: string;
  step: PartStep;
  status: PartStatus;
};

export type Run = {
  id: string;
  deployment_id: string;
  action: RunAction;
  status: RunStatus;
  diagram_version: number | null;
  // The deployment's config_version the run was built from; null for a run
  // from before B recorded it.
  config_version?: number | null;
  failure: string | null;
  created_at: string;
  started_at: string | null;
  finished_at: string | null;
  // Each cloud of a split run, in the order the run takes them; null for a
  // run on one cloud.
  parts?: PartProgress[] | null;
};

// Blocks of a deployment placed on another of the organization's
// connections, with that cloud's own settings (ADR-020). Every other block
// runs on the deployment's own connection.
export type Placement = {
  connection_id: string;
  connection_name: string;
  connector: string;
  region: string;
  node_ids: string[] | null;
  settings: Record<string, unknown> | null;
};

// What create and PATCH take for one placement.
export type PlacementInput = {
  connection_id: string;
  node_ids: string[];
  settings: Record<string, unknown>;
};

export type Deployment = {
  id: string;
  diagram_id: string;
  connection_id: string;
  settings: Record<string, unknown> | null;
  exclusions: string[] | null;
  placements?: Placement[] | null;
  status: DeploymentStatus;
  // Counts the saved changes to its settings, left-out blocks and placements.
  config_version?: number | null;
  created_at: string;
  updated_at: string;
};

// One row of a diagram's deploy panel.
export type DeploymentListItem = Deployment & {
  connector: string;
  connection_name: string;
  region: string;
  latest_run: Run | null;
  // The latest completed apply, whose results "What was built" shows.
  last_apply?: Run | null;
  // The version of the diagram as saved now.
  diagram_version?: number | null;
};

// One row of an organization's deployments: a diagram's deploy panel row,
// with the project its diagram is in. Mirrors orgDeploymentListItem in
// backend/adapter/internal/api/handlers.go.
export type OrgDeploymentListItem = DeploymentListItem & {
  project_id: string;
};

const holdsLastApply = (item: Pick<DeploymentListItem, "status" | "last_apply">) =>
  item.status !== "destroyed" && item.status !== "new" && !!item.last_apply;

// diagramNotDeployed reports whether the saved diagram is newer than the one
// the deployment's last successful apply built, while it still holds what
// that apply built.
export function diagramNotDeployed(item: Pick<DeploymentListItem, "status" | "last_apply" | "diagram_version">): boolean {
  const built = item.last_apply?.diagram_version;
  const saved = item.diagram_version;
  return holdsLastApply(item) && typeof built === "number" && typeof saved === "number" && saved > built;
}

// changesNotDeployed reports whether the saved diagram, or the deployment's
// own settings, left-out blocks or placements, changed since its last
// successful apply, while it still holds what that apply built. An apply from
// before B recorded config_version counts as 0, where every deployment's
// count started, so a change saved since still shows.
export function changesNotDeployed(item: Pick<DeploymentListItem, "status" | "last_apply" | "diagram_version" | "config_version">): boolean {
  const saved = item.config_version;
  const configChanged = holdsLastApply(item) && typeof saved === "number" && saved > (item.last_apply?.config_version ?? 0);
  return diagramNotDeployed(item) || configChanged;
}

export type Problem = {
  node_id?: string;
  edge_source?: string;
  edge_target?: string;
  field?: string;
  code: string;
  message: string;
};

// connectors.CodeCost: a note that names what a deployment costs beyond its
// blocks. The deploy confirmation says it; the compatibility lines do not.
export const COST_NOTE = "cost";

// Problems stop a run; notes do not.
export type Report = {
  ready: boolean;
  problems: Problem[] | null;
  notes: Problem[] | null;
};

// One cloud of a split deployment; position is its place in a run's order.
export type CloudReport = Report & {
  connection_id: string;
  connection_name: string;
  connector: string;
  position: number;
};

// A split deployment's report also lists each cloud; its own problems and
// notes then gather every cloud's, and the cross-cloud ones.
export type Compatibility = Report & {
  clouds?: CloudReport[] | null;
};

export type CreateDeploymentInput = {
  diagram_id: string;
  connection_id: string;
  settings: Record<string, unknown>;
  exclusions: string[];
  placements?: PlacementInput[];
};

// PATCH replaces whichever of the three it holds. B refuses new placements
// (409 deployment_live) while the deployment may hold resources.
export type UpdateDeploymentInput = {
  settings?: Record<string, unknown>;
  exclusions?: string[];
  placements?: PlacementInput[];
};

export type PreviewFiles = { files: Record<string, string> | null };

export const isActive = (run: Run | null | undefined) => !!run && (run.status === "queued" || run.status === "running");
