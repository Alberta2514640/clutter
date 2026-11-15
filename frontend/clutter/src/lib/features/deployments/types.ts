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
  created_at: string;
  updated_at: string;
};

// One row of a diagram's deploy panel.
export type DeploymentListItem = Deployment & {
  connector: string;
  connection_name: string;
  region: string;
  latest_run: Run | null;
};

export type Problem = {
  node_id?: string;
  edge_source?: string;
  edge_target?: string;
  field?: string;
  code: string;
  message: string;
};

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
