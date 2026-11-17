// What the deploy confirmation says before anything is built: one line of
// what each cloud builds, and what the clouds said it costs beyond the
// blocks (B's notes with code "cost", connectors.CodeCost).

// Relative imports: Vitest runs this file without the "@/" alias.
import { joinAnd } from "../../text";
import { countedKinds } from "../diagram/summary";
import type { BlockRef } from "./compatibility";
import { blockCloud } from "./placements";
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
  const ps = placements ?? [];
  const skip = new Set(exclusions);
  const kept = blocks.filter((b) => !skip.has(b.id));
  if (kept.length === 0) return "Nothing to build.";
  const names = new Map([[primary.id, primary.name], ...ps.map((p) => [p.connection_id, p.connection_name] as [string, string])]);
  const order = [primary.id, ...ps.map((p) => p.connection_id)];
  const parts = order.flatMap((id) => {
    const here = kept.filter((b) => blockCloud(primary.id, ps, b.id) === id);
    if (here.length === 0) return [];
    return [`${joinAnd(countedKinds(here, titleOf).map((g) => g.label))} on ${names.get(id) ?? id}`];
  });
  return `Builds ${parts.join(", and ")}.`;
}

// costLines are the clouds' cost notes, each once.
export function costLines(report: Compatibility | null | undefined): string[] {
  return [...new Set((report?.notes ?? []).filter((n) => n.code === COST_NOTE).map((n) => n.message))];
}
