// Which cloud each block of a deployment runs on (ADR-020): the deployment's
// own connection, or another of the organization's connections it is placed
// on. B owns the rules (it checks every connection belongs to the
// organization, refuses an edge that cannot cross clouds, and orders the
// clouds); this only builds the change the user asks for and words B's
// answers.

// Relative imports: Vitest runs this file without the "@/" alias.
import type { Connection } from "../connections/types";
import type { BlockRef } from "./compatibility";
import type { CloudReport, Compatibility, PartProgress, Placement, PlacementInput, Problem } from "./types";

// blockCloud is the connection a block runs on: its placement's, or the
// deployment's own.
export function blockCloud(primaryConnectionId: string, placements: Placement[], blockId: string): string {
  return placements.find((p) => (p.node_ids ?? []).includes(blockId))?.connection_id ?? primaryConnectionId;
}

// cloudsLine words where a deployment builds, for its card's header: its own
// cloud and region ("Amazon Web Services, ca-central-1"), and for a split
// deployment every cloud it builds on ("OpenStack (RegionOne) and Amazon Web
// Services (ca-central-1)"). A placement that holds no block is not a cloud
// the deployment builds on.
export function cloudsLine(primary: { title: string; region: string }, placements: Placement[] | null | undefined, titleOf: (connector: string) => string): string {
  const others = (placements ?? []).filter((p) => (p.node_ids ?? []).length > 0);
  if (others.length === 0) return primary.region ? `${primary.title}, ${primary.region}` : primary.title;
  const parts = [primary, ...others.map((p) => ({ title: titleOf(p.connector), region: p.region }))].map((c) =>
    c.region ? `${c.title} (${c.region})` : c.title,
  );
  return `${parts.slice(0, -1).join(", ")} and ${parts[parts.length - 1]}`;
}

// asInputs turns placements as B lists them into what B takes back.
export function asInputs(placements: Placement[]): PlacementInput[] {
  return placements.map((p) => ({ connection_id: p.connection_id, node_ids: [...(p.node_ids ?? [])], settings: { ...(p.settings ?? {}) } }));
}

// movePlacement places one block on a connection (the primary takes it
// back), keeping only blocks still in the diagram, each cloud's settings, and
// clouds that still hold a block.
export function movePlacement(
  primaryConnectionId: string,
  placements: Placement[],
  blocks: BlockRef[],
  blockId: string,
  connectionId: string,
): PlacementInput[] {
  const ids = new Set(blocks.map((b) => b.id));
  const out = asInputs(placements).map((p) => ({ ...p, node_ids: p.node_ids.filter((n) => ids.has(n) && n !== blockId) }));
  if (connectionId !== primaryConnectionId && ids.has(blockId)) {
    const target = out.find((p) => p.connection_id === connectionId);
    if (target) target.node_ids.push(blockId);
    else out.push({ connection_id: connectionId, node_ids: [blockId], settings: {} });
  }
  return out.filter((p) => p.node_ids.length > 0).map((p) => ({ ...p, node_ids: [...p.node_ids].sort() }));
}

// withPlacementSettings replaces one placed cloud's settings.
export function withPlacementSettings(placements: Placement[], connectionId: string, settings: Record<string, unknown>): PlacementInput[] {
  return asInputs(placements).map((p) => (p.connection_id === connectionId ? { ...p, settings } : p));
}

// cloudChoices are the connections a block may run on: the deployment's own
// first, then the organization's other working connections whose cloud this
// server offers. A connection a block already sits on stays listed even when
// its last check failed, so the select can show it.
export function cloudChoices(
  primary: Connection | undefined,
  connections: Connection[],
  offered: (connector: string) => boolean,
  inUse: ReadonlySet<string>,
): Connection[] {
  const others = connections.filter(
    (c) => c.id !== primary?.id && offered(c.connector) && (c.status === "working" || inUse.has(c.id)),
  );
  return primary ? [primary, ...others] : others;
}

// placementReason is why blocks cannot move between clouds now, or null when
// they can. B refuses the same (409 deployment_live).
export function placementReason(status: string, running: boolean): string | null {
  if (running) return "A run is in progress. Wait for it to finish.";
  if (status !== "new" && status !== "destroyed") {
    return "Blocks move between clouds only while the deployment holds nothing. Destroy it first, then choose where each block runs.";
  }
  return null;
}

// Problem codes B gives a split deployment (backend/adapter/internal/split).
const KEY_NOTE = "cross_cloud_key";
const ADDRESS_NOTE = "cross_cloud_address";

// crossNotes are B's sentences about what crosses clouds, permission edges
// first: "Clutter creates a key that can only read files and puts it on web. ..."
export function crossNotes(report: Compatibility): { keys: string[]; addresses: string[] } {
  const notes = report.notes ?? [];
  return {
    keys: notes.filter((n) => n.code === KEY_NOTE).map((n) => n.message),
    addresses: notes.filter((n) => n.code === ADDRESS_NOTE).map((n) => n.message),
  };
}

const same = (a: Problem, b: Problem) =>
  a.code === b.code && a.node_id === b.node_id && a.edge_source === b.edge_source && a.edge_target === b.edge_target && a.field === b.field;

// splitOnly is what a split report says beyond its clouds' own reports: the
// problems and notes about the split itself (an edge that cannot cross, a
// circle between clouds), without the cross-cloud notes crossNotes words.
export function splitOnly(report: Compatibility): { problems: Problem[]; notes: Problem[] } {
  const clouds: CloudReport[] = report.clouds ?? [];
  const inCloud = (p: Problem, pick: (c: CloudReport) => Problem[] | null) => clouds.some((c) => (pick(c) ?? []).some((q) => same(p, q)));
  return {
    problems: (report.problems ?? []).filter((p) => !inCloud(p, (c) => c.problems)),
    notes: (report.notes ?? []).filter((n) => n.code !== KEY_NOTE && n.code !== ADDRESS_NOTE && !inCloud(n, (c) => c.notes)),
  };
}

// orderedClouds lists a split report's clouds in the order a run takes them.
export function orderedClouds(report: Compatibility): CloudReport[] {
  return [...(report.clouds ?? [])].sort((a, b) => a.position - b.position);
}

const STEP: Record<PartProgress["step"], string> = { plan: "plan", apply: "deploy", destroy: "destroy", rollback: "roll-back" };

const STATUS: Record<PartProgress["status"], string> = {
  queued: "waiting",
  running: "running",
  completed: "finished",
  failed: "failed",
  skipped: "not started, because an earlier cloud failed",
  rolled_back: "rolled back",
};

export type PartLine = { connectionId: string; text: string; status: PartProgress["status"] };

// partLines is one line per cloud of a split run: "lab: deploy finished".
export function partLines(parts: PartProgress[] | null | undefined): PartLine[] {
  return (parts ?? []).map((p) => ({
    connectionId: p.connection_id,
    status: p.status,
    text: p.status === "rolled_back" ? `${p.name}: rolled back` : `${p.name}: ${STEP[p.step] ?? p.step} ${STATUS[p.status] ?? p.status}`,
  }));
}
