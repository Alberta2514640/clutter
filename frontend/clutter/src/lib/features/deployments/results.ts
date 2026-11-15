// "What was built": after an apply, each block, what it became on its cloud,
// and the results its connector returned (a server's addresses, an HTTP API's
// address, a bucket's or a table's name). B writes the results into the run's
// log after its own apply line, one per line:
//
//   Apply complete: 3 blocks built or changed.
//   Result: web private_address 10.0.0.5
//
// (jobs.ResultLinePrefix in backend/adapter/internal/jobs; a split run writes
// them after each cloud's "<cloud>: apply complete, ..." line.) Only lines
// after one of B's apply lines count, so nothing earlier in the log can pass
// for a result.

// Relative imports: Vitest runs this file without the "@/" alias.
import type { BlockRef } from "./compatibility";

export type BlockResult = { name: string; value: string };

const APPLY_LINE = /^(?:Apply complete: |.+: apply complete, )/;
const RESULT_LINE = /^Result: ([A-Za-z0-9_-]{1,64}) ([a-z_]{1,40}) (\S{1,1024})$/;

// parseResults reads the results from a run's log, by node id. A later line
// for the same block and name replaces an earlier one.
export function parseResults(log: string): Map<string, BlockResult[]> {
  const out = new Map<string, BlockResult[]>();
  let applied = false;
  for (const raw of log.split("\n")) {
    const line = raw.trim();
    if (APPLY_LINE.test(line)) {
      applied = true;
      continue;
    }
    if (!applied) continue;
    const m = line.match(RESULT_LINE);
    if (!m) continue;
    const [, id, name, value] = m;
    const list = (out.get(id) ?? []).filter((r) => r.name !== name);
    out.set(id, [...list, { name, value }]);
  }
  return out;
}

// What a person reads for each result, in the order they are listed.
const LABELS: Record<string, string> = {
  public_address: "Public address",
  private_address: "Private address",
  url: "Address",
  bucket: "Bucket name",
  table: "Table name",
};
const ORDER = Object.keys(LABELS);

export type ResultItem = { label: string; value: string; href: string | null };

export type ResultRow = { id: string; name: string; kind: string; became: string; items: ResultItem[] };

// An address opens as a link only when it is a plain web address.
const linkFor = (value: string) => (/^https?:\/\/[^\s"'<>]+$/.test(value) ? value : null);

// resultRows is one row per saved block, in the diagram's order: what it
// became (became names the cloud's service for it) and its results. Blocks
// the deployment left out are left out here too.
export function resultRows(
  results: Map<string, BlockResult[]>,
  blocks: BlockRef[],
  became: (block: BlockRef) => string,
  excluded: string[] = [],
): ResultRow[] {
  const skip = new Set(excluded);
  return blocks
    .filter((b) => !skip.has(b.id))
    .map((b) => {
      const items = [...(results.get(b.id) ?? [])]
        .sort((x, y) => rank(x.name) - rank(y.name))
        .map((r) => ({ label: LABELS[r.name] ?? r.name, value: r.value, href: r.name === "url" ? linkFor(r.value) : null }));
      return { id: b.id, name: b.name, kind: b.kind, became: became(b), items };
    });
}

const rank = (name: string) => {
  const i = ORDER.indexOf(name);
  return i < 0 ? ORDER.length : i;
};
