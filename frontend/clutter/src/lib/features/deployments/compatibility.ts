// The compatibility line of one deployment, from B's capability report
// (GET .../compatibility) and the diagram's blocks. B owns the check; this only
// words it and lists the blocks the user can leave out.

// Relative imports: Vitest runs this file without the "@/" alias.
import { joinAnd, plural } from "../../text";
import { manifestOffers } from "../catalog/forms";
import type { Manifest } from "../catalog/types";
import { COST_NOTE, type Compatibility, type Problem } from "./types";

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

const UNSUPPORTED = "unsupported_block";

function where(p: Problem, byId: Map<string, BlockRef>): string {
  const label = (id?: string) => (id ? (byId.get(id)?.name || id) : "");
  if (p.node_id) return `${label(p.node_id)}: `;
  if (p.edge_source && p.edge_target) return `${label(p.edge_source)} to ${label(p.edge_target)}: `;
  if (p.field) return "Settings: ";
  return "";
}

// compatibilityLine words a deployment's report. Exclusions that name a block
// no longer in the diagram are ignored (B reports them as a note). part is
// set for one cloud of a split deployment: its blocks are only some of the
// diagram's, so a ready line says so instead of "every block".
export function compatibilityLine(
  cloudTitle: string,
  report: Compatibility,
  blocks: BlockRef[],
  exclusions: string[],
  kindTitle: (kind: string) => string,
  part = false,
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
    const placed = blocks.filter((b) => !excluded.has(b.id)).length;
    parts.push(
      part
        ? `the ${plural(placed, "block", "blocks")} placed here can be built`
        : excluded.size > 0
          ? `the other blocks can be built here, ${plural(excluded.size, "block", "blocks")} left out`
          : "every block can be built here",
    );
  }

  return {
    ready: report.ready,
    summary: `${cloudTitle}: ${parts.join(", ")}`,
    choices,
    problems: other.map((p) => `${where(p, byId)}${p.message}`),
    // A cost note is said once, in the deploy confirmation (confirm.ts).
    notes: (report.notes ?? []).filter((p) => p.code !== COST_NOTE).map((p) => `${where(p, byId)}${p.message}`),
  };
}

// splitReadyLine is the one line a split deployment shows when every one of
// its clouds can build its blocks and nothing needs a choice: "Every block
// can be built across Local OpenStack and Local AWS". It is null otherwise,
// and the panel shows each cloud's own line.
export function splitReadyLine(lines: CompatibilityLine[], cloudNames: string[], leftOut: number): string | null {
  if (lines.length < 2 || lines.some((l) => !l.ready || l.choices.length > 0 || l.problems.length > 0)) return null;
  const across = joinAnd(cloudNames);
  return leftOut > 0
    ? `The other blocks can be built across ${across}, ${plural(leftOut, "block", "blocks")} left out`
    : `Every block can be built across ${across}`;
}

// What the Block panel shows about one block from a deployment's report:
// problems under the field they name, the block's other problems (its name
// is a field too), and notes such as advice. Cost notes are left to the
// deploy confirmation.
export type BlockIssues = { fields: Map<string, string[]>; general: string[]; notes: string[] };

export function blockIssues(report: Compatibility | null | undefined, id: string, fieldNames: ReadonlySet<string>): BlockIssues {
  const out: BlockIssues = { fields: new Map(), general: [], notes: [] };
  const add = (list: string[], m: string) => {
    if (!list.includes(m)) list.push(m);
  };
  for (const p of report?.problems ?? []) {
    if (p.node_id === id && p.field && fieldNames.has(p.field)) {
      const list = out.fields.get(p.field) ?? [];
      add(list, p.message);
      out.fields.set(p.field, list);
    } else if (p.node_id === id || (!p.node_id && p.edge_source === id)) {
      add(out.general, p.message);
    }
  }
  for (const n of report?.notes ?? []) {
    if (n.node_id === id && n.code !== COST_NOTE) add(out.notes, n.message);
  }
  return out;
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
