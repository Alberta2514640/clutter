// What the deploy confirmation says before anything is built: one line of
// what each cloud builds, and what the clouds said it costs beyond the
// blocks (B's notes with code "cost", connectors.CodeCost).

// Relative imports: Vitest runs this file without the "@/" alias.
import { joinAnd } from "../../text";
import { countedKinds } from "../diagram/summary";
import type { BlockRef } from "./compatibility";
import { blockCloud } from "./placements";
import { builtBlocks } from "./results";
import { COST_NOTE, type Compatibility, type Placement } from "./types";

// deployLine is "Builds 1 server and 1 network on lab." for one cloud, and
// "Builds 1 server on devstack, and 1 bucket on Local AWS." for a
// split deployment. Left-out blocks are not built, so they are not named.
export function deployLine(
  primary: { id: string; name: string },
  placements: Placement[] | null | undefined,
  blocks: BlockRef[],
  exclusions: string[],
  titleOf: (kind: string) => string,
): string {
  return blocksLine("Builds", "Nothing to build.", primary, placements, blocks, exclusions, titleOf);
}

// destroyLine is the destroy confirmation's list, worded as deployLine's:
// "Destroys 1 server and 1 network on lab." It counts what the last
// successful apply built (its log's "Built:" lines), never a block added to
// the diagram since. With no such record (no apply yet, its log not loaded,
// or a run from before B named its blocks) it counts the saved diagram,
// less the left-out blocks, which were never built.
export function destroyLine(
  primary: { id: string; name: string },
  placements: Placement[] | null | undefined,
  blocks: BlockRef[],
  exclusions: string[],
  titleOf: (kind: string) => string,
  lastApplyLog?: string | null,
): string {
  const built = lastApplyLog ? builtBlocks(lastApplyLog, blocks) : null;
  return blocksLine("Destroys", "Nothing is left to destroy.", primary, placements, built ?? blocks, built ? [] : exclusions, titleOf);
}

function blocksLine(
  verb: string,
  none: string,
  primary: { id: string; name: string },
  placements: Placement[] | null | undefined,
  blocks: BlockRef[],
  exclusions: string[],
  titleOf: (kind: string) => string,
): string {
  const ps = placements ?? [];
  const skip = new Set(exclusions);
  const kept = blocks.filter((b) => !skip.has(b.id));
  if (kept.length === 0) return none;
  const names = new Map([[primary.id, primary.name], ...ps.map((p) => [p.connection_id, p.connection_name] as [string, string])]);
  const order = [primary.id, ...ps.map((p) => p.connection_id)];
  const parts = order.flatMap((id) => {
    const here = kept.filter((b) => blockCloud(primary.id, ps, b.id) === id);
    if (here.length === 0) return [];
    return [`${joinAnd(countedKinds(here, titleOf).map((g) => g.label))} on ${names.get(id) ?? id}`];
  });
  return `${verb} ${parts.join(", and ")}.`;
}

// costLines are the clouds' cost notes, each once.
export function costLines(report: Compatibility | null | undefined): string[] {
  return [...new Set((report?.notes ?? []).filter((n) => n.code === COST_NOTE).map((n) => n.message))];
}
