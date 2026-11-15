// What the Connections page says about deleting a connection. B deletes a
// connection with the deployments that use it only while none of them has
// something built in the cloud; otherwise it refuses with 409
// live_deployments (backend/adapter/internal/api/handlers.go, deleteConnection).

// Relative imports: Vitest runs this file without the "@/" alias.
import { AdapterApiError, adapterErrorMessage } from "../../adapterApi";

const LIVE_DEPLOYMENTS = "live_deployments";

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
