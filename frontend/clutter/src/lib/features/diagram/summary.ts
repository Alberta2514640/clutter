// The one-line diagram summary at the bottom of B's canvas ("Builds 2
// servers, 1 network, 1 disk") and the short list it opens to, grouped by
// kind. Plain nouns name the standard blocks; a kind a cloud adds on its own
// is named by its title.

// Relative imports: Vitest runs this file without the "@/" alias.
import { plural } from "../../text";
import { STANDARD_BLOCKS } from "../catalog/types";

// What a person calls each standard block, one and many.
const NOUNS: Record<string, [string, string]> = {
  vm: ["server", "servers"],
  network: ["network", "networks"],
  firewall: ["firewall", "firewalls"],
  volume: ["disk", "disks"],
  bucket: ["bucket", "buckets"],
  function: ["function", "functions"],
  table: ["table", "tables"],
  http_api: ["API", "APIs"],
};

export type SummaryGroup = { kind: string; label: string; names: string[] };

export type DiagramSummary = { line: string; groups: SummaryGroup[] };

const counted = (kind: string, n: number, titleOf: (kind: string) => string) => {
  const [one, many] = NOUNS[kind] ?? [titleOf(kind), titleOf(kind)];
  return `${n} ${n === 1 ? one : many}`;
};

// countedKinds groups the blocks by kind ("2 servers"), standard kinds in
// their fixed order, then the others by title. The deploy confirmation reads
// it too (deployments/confirm.ts).
export function countedKinds(blocks: { kind: string; name: string }[], titleOf: (kind: string) => string): SummaryGroup[] {
  const byKind = new Map<string, string[]>();
  for (const b of blocks) byKind.set(b.kind, [...(byKind.get(b.kind) ?? []), b.name || "(no name)"]);
  const kinds = [
    ...STANDARD_BLOCKS.filter((k) => byKind.has(k)),
    ...[...byKind.keys()].filter((k) => !STANDARD_BLOCKS.includes(k)).sort((a, b) => titleOf(a).localeCompare(titleOf(b))),
  ];
  return kinds.map((kind) => ({ kind, label: counted(kind, byKind.get(kind)!.length, titleOf), names: byKind.get(kind)! }));
}

// diagramSummary is the canvas's one line and the groups it opens to. With a
// deployment chosen it counts only what that deployment builds ("Builds 2
// servers, 1 network"), never a block it leaves out, and names how many it
// leaves out. With none it says what the diagram has ("Has 3 blocks"), since
// nothing says where it would be built.
export function diagramSummary(
  blocks: { id?: string; kind: string; name: string }[],
  titleOf: (kind: string) => string,
  deployment: { excluded: readonly string[] } | null = null,
): DiagramSummary {
  if (blocks.length === 0) return { line: "Nothing to build yet", groups: [] };
  if (!deployment) return { line: `Has ${plural(blocks.length, "block", "blocks")}`, groups: countedKinds(blocks, titleOf) };
  const skip = new Set(deployment.excluded);
  const kept = blocks.filter((b) => !b.id || !skip.has(b.id));
  const left = blocks.length - kept.length;
  const groups = countedKinds(kept, titleOf);
  const built = groups.length === 0 ? "Builds nothing" : `Builds ${groups.map((g) => g.label).join(", ")}`;
  return { line: left > 0 ? `${built}, ${left} left out` : built, groups };
}
