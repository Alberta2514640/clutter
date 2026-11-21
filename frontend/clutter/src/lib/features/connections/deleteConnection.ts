// What the Connections page says about deleting a connection. B deletes a
// connection with the deployments that use it only while none of them has
// something built in the cloud; otherwise it refuses with 409
// live_deployments (backend/adapter/internal/api/handlers.go, deleteConnection).

// Relative imports: Vitest runs this file without the "@/" alias.
import { AdapterApiError, adapterErrorMessage } from "../../adapterApi";
import type { DeploymentStatus } from "../deployments/types";
import { holdsResources } from "../projects/deploymentSummary";

const LIVE_DEPLOYMENTS = "live_deployments";

// A diagram with a deployment on the connection that may still hold
// resources, and the link to its Deploy view.
export type Holder = { diagramId: string; name: string; href: string };

type Row = {
  connection_id: string;
  diagram_id: string;
  project_id: string;
  status: DeploymentStatus;
  placements?: { connection_id: string }[] | null;
};

type ProjectRef = { id: string; diagrams?: { id: string; name: string }[] | null };

// holdingDiagrams is each diagram whose deployment builds on this connection
// (as its own connection or one a block is placed on) and may still hold
// resources, once each, named from the team's projects. Delete stays off
// while there is one, as B would refuse it (409 live_deployments).
export function holdingDiagrams(connectionId: string, rows: readonly Row[], projects: readonly ProjectRef[]): Holder[] {
  const out = new Map<string, Holder>();
  for (const r of rows) {
    const uses = r.connection_id === connectionId || (r.placements ?? []).some((p) => p.connection_id === connectionId);
    if (!uses || !holdsResources(r.status) || out.has(r.diagram_id)) continue;
    const name = projects.find((p) => p.id === r.project_id)?.diagrams?.find((d) => d.id === r.diagram_id)?.name ?? "a diagram";
    out.set(r.diagram_id, { diagramId: r.diagram_id, name, href: `/projects/${r.project_id}/diagram/${r.diagram_id}?panel=deploy` });
  }
  return [...out.values()];
}

// deleteWarning is what the confirm step says, given how many diagrams deploy
// to the connection.
export function deleteWarning(diagrams: number): string {
  if (diagrams === 0) return "No diagram deploys to this connection. Delete it?";
  if (diagrams === 1) {
    return "1 diagram deploys to this connection. If its deployment is built, destroy it first. If not, deleting removes the deployment and changes nothing in your cloud.";
  }
  return `${diagrams} diagrams deploy to this connection. Destroy the deployments that are built first. Deleting removes the rest and changes nothing in your cloud.`;
}

// deleteErrorMessage is why a delete was refused, in the page's own words for
// B's refusal while a deployment is still built.
export function deleteErrorMessage(err: unknown): string {
  if (err instanceof AdapterApiError && err.code === LIVE_DEPLOYMENTS) {
    return "A deployment on this connection is still built. Destroy it on its diagram's Deploy tab first.";
  }
  return adapterErrorMessage(err);
}
