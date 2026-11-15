// The one-line diagram summary at the bottom of B's canvas ("Builds 2
// servers, 1 network, 1 disk") and the short list it opens to, grouped by
// kind. Plain nouns name the standard blocks; a kind a cloud adds on its own
// is named by its title.

// Relative imports: Vitest runs this file without the "@/" alias.
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

// diagramSummary groups the blocks by kind, standard kinds in their fixed
// order, then the others by title.
export function diagramSummary(blocks: { kind: string; name: string }[], titleOf: (kind: string) => string): DiagramSummary {
  const byKind = new Map<string, string[]>();
  for (const b of blocks) byKind.set(b.kind, [...(byKind.get(b.kind) ?? []), b.name || "(no name)"]);
  const kinds = [
    ...STANDARD_BLOCKS.filter((k) => byKind.has(k)),
    ...[...byKind.keys()].filter((k) => !STANDARD_BLOCKS.includes(k)).sort((a, b) => titleOf(a).localeCompare(titleOf(b))),
  ];
  const groups = kinds.map((kind) => ({ kind, label: counted(kind, byKind.get(kind)!.length, titleOf), names: byKind.get(kind)! }));
  return { line: groups.length === 0 ? "Nothing to build yet" : `Builds ${groups.map((g) => g.label).join(", ")}`, groups };
}
