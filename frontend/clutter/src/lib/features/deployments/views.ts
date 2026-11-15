// The "View as" logic of B's editor: which clouds the switch lists, and how a
// block looks in each view. A diagram is drawn once and deployed to many clouds
// (ADR-014); a cloud view only shows the same diagram as that cloud builds it,
// so editing is the same in every view.

// Relative imports: Vitest runs this file without the "@/" alias.
import { blockLook, cloudLook, type CloudLook } from "../catalog/display";
import type { CatalogEntry, Manifest } from "../catalog/types";
import type { Connection } from "../connections/types";
import type { DeploymentListItem } from "./types";

// The view that shows plain building blocks.
export const BLOCKS_VIEW = "blocks";

// One of this diagram's deployments as one cloud sees it. A deployment on
// one cloud runs every block there. A split one runs some blocks on each of
// its clouds (ADR-020): on its own cloud, every block not placed on another
// cloud (away); on a placed cloud, only the blocks placed there (only).
export type CloudDeployment = {
  id: string;
  connectionName: string;
  exclusions: string[];
  only: string[] | null;
  away: string[];
};

// runsHere reports whether a deployment runs a block on this cloud.
export const runsHere = (d: CloudDeployment, blockId: string) => (d.only ? d.only.includes(blockId) : !d.away.includes(blockId));

// One cloud the switch offers.
export type CloudView = CloudLook & {
  manifest: Manifest;
  // This diagram's deployments that build on the cloud. Empty when the
  // organization has a connection to the cloud but no deployment builds there.
  deployments: CloudDeployment[];
};

// viewClouds lists every cloud the diagram has a deployment on or the
// organization has a connection to, in catalog order. A cloud the catalog no
// longer offers has no manifest to show, so it is left out.
export function viewClouds(catalog: CatalogEntry[], deployments: DeploymentListItem[], connections: Connection[]): CloudView[] {
  const deployed = new Map<string, CloudView["deployments"]>();
  const add = (connector: string, d: CloudDeployment) => deployed.set(connector, [...(deployed.get(connector) ?? []), d]);
  for (const d of deployments) {
    const placed = (d.placements ?? []).filter((p) => (p.node_ids ?? []).length > 0);
    const exclusions = d.exclusions ?? [];
    // Blocks placed on a connection of another cloud do not run on this one.
    const away = placed.filter((p) => p.connector !== d.connector).flatMap((p) => p.node_ids ?? []);
    add(d.connector, { id: d.id, connectionName: d.connection_name, exclusions, only: null, away });
    const others = new Map<string, { names: string[]; only: string[] }>();
    for (const p of placed) {
      if (p.connector === d.connector) continue;
      const o = others.get(p.connector) ?? { names: [], only: [] };
      o.names.push(p.connection_name);
      o.only.push(...(p.node_ids ?? []));
      others.set(p.connector, o);
    }
    for (const [connector, o] of others) add(connector, { id: d.id, connectionName: o.names.join(", "), exclusions, only: o.only, away: [] });
  }
  const connected = new Set(connections.map((c) => c.connector));
  return catalog
    .filter(({ manifest }) => deployed.has(manifest.connector) || connected.has(manifest.connector))
    .map(({ manifest }) => ({ ...cloudLook(manifest), manifest, deployments: deployed.get(manifest.connector) ?? [] }));
}

// resolveView keeps a remembered choice while its cloud is still listed, and
// falls back to the blocks view otherwise.
export function resolveView(choice: string | null | undefined, views: CloudView[]): string {
  if (choice && views.some((v) => v.connector === choice)) return choice;
  return BLOCKS_VIEW;
}

export type BlockRefLite = { id: string; kind: string };

export type BlockStatus = "builds" | "not-offered" | "elsewhere" | "left-out" | "partly-left-out";

export type CloudBlockState = {
  status: BlockStatus;
  // A block the cloud will not build in any of this diagram's deployments.
  dimmed: boolean;
  // A short visible label, or null when nothing needs saying.
  label: string | null;
  // A full sentence for the tooltip and screen readers.
  detail: string;
};

// cloudBlockState is how one cloud treats one block of this diagram.
export function cloudBlockState(view: CloudView, block: BlockRefLite, blockTitle: string): CloudBlockState {
  const look = blockLook(view.manifest, block.kind, blockTitle);
  const here = view.deployments.filter((d) => runsHere(d, block.id));
  // Placed on another cloud by every deployment that builds here.
  if (view.deployments.length > 0 && here.length === 0) {
    return { status: "elsewhere", dimmed: true, label: "On another cloud", detail: `Runs on another cloud, not on ${view.shortName}.` };
  }
  if (!look.offered) {
    return { status: "not-offered", dimmed: true, label: `Not on ${view.shortName}`, detail: `${view.title} does not offer ${blockTitle} blocks.` };
  }
  const total = here.length;
  const out = here.filter((d) => d.exclusions.includes(block.id));
  if (total > 0 && out.length === total) {
    return {
      status: "left-out",
      dimmed: true,
      label: "Left out",
      detail: total === 1 ? `Left out of the deployment to ${out[0].connectionName} (${view.shortName}).` : `Left out of every ${view.shortName} deployment.`,
    };
  }
  if (out.length > 0) {
    return {
      status: "partly-left-out",
      dimmed: false,
      label: `Left out of ${out.length} of ${total}`,
      detail: `Left out of ${out.map((d) => d.connectionName).join(", ")} (${view.shortName}); the other ${view.shortName} deployments build it as ${look.name}.`,
    };
  }
  return {
    status: "builds",
    dimmed: false,
    label: null,
    detail: `${view.shortName} ${total > 0 ? "builds" : "can build"} it as ${look.name}.`,
  };
}

export type NodeBadge = { cloud: CloudLook; on: boolean; detail: string };

// How a node looks in the current view.
export type NodeLook = {
  subtitle: string;
  // The cloud's icon for the block, or null for the neutral icon.
  iconSrc: string | null;
  dimmed: boolean;
  status: string | null;
  statusDetail: string | null;
  // In the blocks view, one badge per cloud the diagram deploys to.
  badges: NodeBadge[];
};

export function nodeLook(view: string, views: CloudView[], block: BlockRefLite, blockTitle: string): NodeLook {
  const current = view === BLOCKS_VIEW ? undefined : views.find((v) => v.connector === view);
  if (current) {
    const look = blockLook(current.manifest, block.kind, blockTitle);
    const state = cloudBlockState(current, block, blockTitle);
    return { subtitle: look.name, iconSrc: look.iconSrc, dimmed: state.dimmed, status: state.label, statusDetail: state.detail, badges: [] };
  }
  // One badge per cloud a deployment would build the block on: on where it is
  // built, off where it is left out or cannot be built. A cloud the block is
  // placed away from (it runs on another cloud of the same deployment) shows
  // no badge.
  const badges = views
    .filter((v) => v.deployments.length > 0)
    .map((v) => ({ cloud: v, state: cloudBlockState(v, block, blockTitle) }))
    .filter((s) => s.state.status !== "elsewhere")
    .map((s) => ({ cloud: s.cloud, on: !s.state.dimmed, detail: s.state.detail }));
  return { subtitle: blockTitle, iconSrc: null, dimmed: false, status: null, statusDetail: null, badges };
}

const plural = (n: number, one: string, many: string) => `${n} ${n === 1 ? one : many}`;

// viewSummary is the one line a cloud view shows above the canvas.
export function viewSummary(view: CloudView, blocks: BlockRefLite[], blockTitle: (kind: string) => string): string {
  let notOffered = 0;
  let elsewhere = 0;
  let leftOut = 0;
  let partly = 0;
  for (const b of blocks) {
    const s = cloudBlockState(view, b, blockTitle(b.kind)).status;
    if (s === "not-offered") notOffered++;
    else if (s === "elsewhere") elsewhere++;
    else if (s === "left-out") leftOut++;
    else if (s === "partly-left-out") partly++;
  }
  const parts: string[] = [];
  if (notOffered > 0) parts.push(`${plural(notOffered, "block", "blocks")} not on ${view.shortName}`);
  if (elsewhere > 0) parts.push(`${plural(elsewhere, "block runs", "blocks run")} on another cloud`);
  if (leftOut > 0) parts.push(`${plural(leftOut, "block", "blocks")} left out`);
  if (partly > 0) parts.push(`${plural(partly, "block", "blocks")} left out of some deployments`);
  const tail = parts.length > 0 ? parts.join(", ") : blocks.length > 0 ? `${view.shortName} builds every block` : "no blocks yet";
  const where = view.deployments.length === 0 ? " (no deployment yet)" : "";
  return `Viewing as ${view.title}${where}: ${tail}.`;
}
