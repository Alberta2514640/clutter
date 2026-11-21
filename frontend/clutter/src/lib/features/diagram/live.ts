// Which blocks on B's canvas are Live: those the shown deployment's last
// successful apply built, read from that apply's "Built:" lines
// (deployments/results.ts), each on the connection it runs on. A block drawn
// since that apply is not built yet; one the deployment leaves out is
// neither. A run from before B wrote those lines names no blocks: while the
// saved diagram and the deployment's settings are still the ones it built,
// its blocks stand in; once either has changed, no block is marked either way.

// Relative imports: Vitest runs this file without the "@/" alias.
import { builtBlocks } from "../deployments/results";
import { changesNotDeployed, type DeploymentListItem } from "../deployments/types";

export type LiveMarkers = {
  // "Live on <connection>", by node id.
  live: Map<string, string>;
  // The blocks on the canvas the apply did not build.
  notBuilt: Set<string>;
};

export const NO_MARKERS: LiveMarkers = { live: new Map(), notBuilt: new Set() };

type Shown = Pick<DeploymentListItem, "status" | "last_apply" | "diagram_version" | "config_version" | "exclusions" | "placements" | "connection_name">;

// liveMarkers reads the markers from the deployment, its last apply's log
// (null while it loads), the saved diagram's node ids and the canvas's.
export function liveMarkers(d: Shown | null, log: string | null, savedIds: string[], canvasIds: string[]): LiveMarkers {
  if (!d || !d.last_apply || (d.status !== "live" && d.status !== "failed") || log === null) return NO_MARKERS;
  const left = new Set(d.exclusions ?? []);
  const named = builtBlocks(log, []);
  let built: Set<string>;
  if (named) built = new Set(named.map((b) => b.id));
  else if (!changesNotDeployed(d)) built = new Set(savedIds.filter((id) => !left.has(id)));
  else return NO_MARKERS;

  const live = new Map<string, string>();
  const notBuilt = new Set<string>();
  for (const id of canvasIds) {
    if (built.has(id)) {
      const placed = (d.placements ?? []).find((p) => (p.node_ids ?? []).includes(id));
      live.set(id, `Live on ${placed?.connection_name ?? d.connection_name}`);
    } else if (!left.has(id)) {
      notBuilt.add(id);
    }
  }
  return { live, notBuilt };
}
