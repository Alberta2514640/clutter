// The compatibility line of one deployment, from B's capability report
// (GET .../compatibility) and the diagram's blocks. B owns the check; this only
// words it and lists the blocks the user can leave out.

// Relative imports: Vitest runs this file without the "@/" alias.
import { manifestOffers } from "../catalog/forms";
import type { Manifest } from "../catalog/types";
import type { Compatibility, Problem } from "./types";

export type BlockRef = { id: string; kind: string; name: string };

// A block the user may leave out of, or put back into, this deployment.
export type ExclusionChoice = {
  id: string;
  name: string;
  kindTitle: string;
  supported: boolean;
  excluded: boolean;
};

export type CompatibilityLine = {
  ready: boolean;
  summary: string;
  choices: ExclusionChoice[];
  // Problems other than unsupported blocks, worded with the block's name.
  problems: string[];
  notes: string[];
};

const plural = (n: number, one: string, many: string) => `${n} ${n === 1 ? one : many}`;

const UNSUPPORTED = "unsupported_block";

function where(p: Problem, byId: Map<string, BlockRef>): string {
  const label = (id?: string) => (id ? (byId.get(id)?.name || id) : "");
  if (p.node_id) return `${label(p.node_id)}: `;
  if (p.edge_source && p.edge_target) return `${label(p.edge_source)} to ${label(p.edge_target)}: `;
  if (p.field) return "Settings: ";
  return "";
}

// compatibilityLine words a deployment's report. Exclusions that name a block
// no longer in the diagram are ignored (B reports them as a note).
export function compatibilityLine(
  cloudTitle: string,
  report: Compatibility,
  blocks: BlockRef[],
  exclusions: string[],
  kindTitle: (kind: string) => string,
): CompatibilityLine {
  const byId = new Map(blocks.map((b) => [b.id, b]));
  const problems = report.problems ?? [];
  const unsupportedIds = new Set(problems.filter((p) => p.code === UNSUPPORTED && p.node_id).map((p) => p.node_id!));
  const excluded = new Set(exclusions.filter((id) => byId.has(id)));

  const choices: ExclusionChoice[] = blocks
    .filter((b) => unsupportedIds.has(b.id) || excluded.has(b.id))
    .map((b) => ({
      id: b.id,
      name: b.name,
      kindTitle: kindTitle(b.kind),
      supported: !unsupportedIds.has(b.id),
      excluded: excluded.has(b.id),
    }));

  const other = problems.filter((p) => !(p.code === UNSUPPORTED && p.node_id));
  const kinds = [...new Set(blocks.filter((b) => unsupportedIds.has(b.id)).map((b) => kindTitle(b.kind)))];

  const parts: string[] = [];
  if (unsupportedIds.size > 0) parts.push(`${plural(unsupportedIds.size, "block", "blocks")} not supported (${kinds.join(", ")})`);
  if (other.length > 0) parts.push(plural(other.length, unsupportedIds.size > 0 ? "other problem" : "problem", unsupportedIds.size > 0 ? "other problems" : "problems"));
  // "Ready" next to "Not deployed" read as deployed, so the line says what
  // it checked: that the cloud can build the blocks.
  if (report.ready) {
    parts.push(excluded.size > 0 ? `the other blocks can be built here, ${plural(excluded.size, "block", "blocks")} left out` : "every block can be built here");
  }

  return {
    ready: report.ready,
    summary: `${cloudTitle}: ${parts.join(", ")}`,
    choices,
    problems: other.map((p) => `${where(p, byId)}${p.message}`),
    notes: (report.notes ?? []).map((p) => `${where(p, byId)}${p.message}`),
  };
}

// problemsByBlock maps each block of a deployment's report to the messages
// of its problems, for the badge on the block: a block's own, and a line's on
// the block it starts from. B's messages say what is wrong and how to fix it.
// Notes are not problems and get no badge.
export function problemsByBlock(report: Compatibility | null | undefined): Map<string, string[]> {
  const out = new Map<string, string[]>();
  for (const p of report?.problems ?? []) {
    const id = p.node_id || p.edge_source;
    if (!id) continue;
    const list = out.get(id) ?? [];
    if (!list.includes(p.message)) list.push(p.message);
    out.set(id, list);
  }
  return out;
}

// unsupportedBlocks is what a cloud cannot build, before a deployment exists:
// the blocks whose kind its manifest does not offer.
export function unsupportedBlocks(manifest: Manifest, blocks: BlockRef[]): BlockRef[] {
  return blocks.filter((b) => !manifestOffers(manifest, b.kind));
}

// nextExclusions toggles one block, keeping only blocks still in the diagram.
export function nextExclusions(current: string[], blocks: BlockRef[], id: string, exclude: boolean): string[] {
  const ids = new Set(blocks.map((b) => b.id));
  const kept = current.filter((x) => ids.has(x) && x !== id);
  return exclude && ids.has(id) ? [...kept, id] : kept;
}
