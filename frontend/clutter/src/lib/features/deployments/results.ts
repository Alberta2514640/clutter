// "What was built": after an apply, each block, what it became on its cloud,
// and the results its connector returned (a server's addresses, an HTTP API's
// address, a bucket's or a table's name). B writes the results into the run's
// log after its own apply line, one per line:
//
//   Apply complete: 3 blocks built or changed.
//   Result: web private_address 10.0.0.5
//
// (jobs.ResultLinePrefix in backend/adapter/internal/jobs; a split run writes
// them after each cloud's "<cloud>: apply complete, ..." line.) Before its
// results, B names each block that apply's deployment holds, as it built it:
//
//   Built: web vm web
//
// (jobs.BuiltLinePrefix), so "What was built" never takes a name the diagram
// was given after the deploy. A run from before those lines has none.
// Only lines after one of B's apply lines count, so nothing earlier in the
// log can pass for a result or a block.

// Relative imports: Vitest runs this file without the "@/" alias.
import { joinAnd } from "../../text";
import { VM_KIND } from "../diagram/blocksDocument";
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

// The block kinds and names follow B's own rules (connectors.BlockKind and
// the capability check's name pattern); the id is a node id.
const BUILT_LINE = /^Built: ([A-Za-z0-9_-]{1,64}) ([a-z][a-z0-9_]{0,39}) ([a-z][a-z0-9_-]{0,63})$/;

// builtBlocks is the blocks a run's apply built, as it built them: each
// "Built:" line's id, kind and name, listed in the order of the saved
// diagram's blocks, then any the diagram no longer holds in log order. It is
// null for a log with no such line (a run from before B wrote them), and the
// caller then falls back to the saved diagram's blocks.
export function builtBlocks(log: string, saved: BlockRef[]): BlockRef[] | null {
  const byId = new Map<string, BlockRef>();
  let applied = false;
  for (const raw of log.split("\n")) {
    const line = raw.trim();
    if (APPLY_LINE.test(line)) {
      applied = true;
      continue;
    }
    if (!applied) continue;
    const m = line.match(BUILT_LINE);
    if (m) byId.set(m[1], { id: m[1], kind: m[2], name: m[3] });
  }
  if (byId.size === 0) return null;
  const order = new Map(saved.map((b, i) => [b.id, i]));
  const rank = (b: BlockRef) => order.get(b.id) ?? order.size;
  return [...byId.values()].map((b, i) => ({ b, i })).sort((x, y) => rank(x.b) - rank(y.b) || x.i - y.i).map(({ b }) => b);
}

// What a person reads for each result, in the order they are listed. The
// function's "code" result is not a value to show: leadLines words it.
const LABELS: Record<string, string> = {
  public_address: "Public address",
  private_address: "Private address",
  url: "Address",
  bucket: "Bucket name",
  table: "Table name",
};

// connectors.ResultCode and connectors.ResultCodeSample: a function that still
// runs Clutter's sample code, which answers every request.
const CODE = "code";
const SAMPLE = "sample";
// connectors.ResultSimulated: a VM built on a cloud emulator (LocalStack). It
// has addresses but never boots, so it has nothing to open.
const SIMULATED = "simulated";
// Results that say something about a block rather than hold a value to show.
const NOT_VALUES = new Set([CODE, SIMULATED]);
const ORDER = Object.keys(LABELS);

const isSimulated = (results: Map<string, BlockResult[]>, id: string) => (results.get(id) ?? []).some((r) => r.name === SIMULATED);

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
        .filter((r) => !NOT_VALUES.has(r.name))
        .sort((x, y) => rank(x.name) - rank(y.name))
        .map((r) => ({ label: LABELS[r.name] ?? r.name, value: r.value, href: r.name === "url" ? linkFor(r.value) : null }));
      return { id: b.id, name: b.name, kind: b.kind, became: became(b), items };
    });
}

const rank = (name: string) => {
  const i = ORDER.indexOf(name);
  return i < 0 ? ORDER.length : i;
};

// What one result says in the shown log: "public address 203.0.113.9".
function resultText(name: string, value: string): string {
  if (name === CODE) return value === SAMPLE ? "answers with a sample until you upload your code" : `code ${value}`;
  if (name === SIMULATED) return "simulated; it does not boot";
  const label = LABELS[name] ?? name.replaceAll("_", " ");
  return `${label.charAt(0).toLowerCase()}${label.slice(1)} ${value}`;
}

// readableLog is a run's log as the deploy panel shows, copies and downloads
// it. B's "Built:" and "Result:" lines, which parseResults and builtBlocks
// read back, become plain lines that name each block as the diagram does
// ("Built server (VM)", "server: public address 203.0.113.9"). kindTitle
// names a block kind ("VM"). Every other line stays as it was written.
export function readableLog(log: string, kindTitle: (kind: string) => string = (k) => k): string {
  const names = new Map<string, string>();
  let applied = false;
  return log
    .split("\n")
    .map((raw) => {
      const line = raw.trim();
      if (APPLY_LINE.test(line)) {
        applied = true;
        return raw;
      }
      if (!applied) return raw;
      const built = line.match(BUILT_LINE);
      if (built) {
        names.set(built[1], built[3]);
        return `Built ${built[3]} (${kindTitle(built[2])})`;
      }
      const result = line.match(RESULT_LINE);
      if (result) return `${names.get(result[1]) ?? result[1]}: ${resultText(result[2], result[3])}`;
      return raw;
    })
    .join("\n");
}

// One plain line per result a person can use, before any detail: a public
// server's address (a website when it serves the web on port 80), a web API's
// address. A private address is a detail, never a lead line.
export type LeadLine =
  | { kind: "website"; id: string; label: string; url: string }
  | { kind: "server"; id: string; label: string; address: string }
  | { kind: "api"; id: string; label: string; url: string };

const BASE_LABEL: Record<LeadLine["kind"], string> = { website: "Your website", server: "Your server", api: "Your API" };

// leadLines reads the results in the diagram's order. web says whether a
// server serves the web on port 80 (servesWeb). A simulated server has no
// line: it never boots (simulatedNote says so). When two lines share a label,
// each names its block: "Your website (shop)".
export function leadLines(results: Map<string, BlockResult[]>, blocks: BlockRef[], web: ReadonlySet<string>, excluded: string[] = []): LeadLine[] {
  const skip = new Set(excluded);
  const out: (LeadLine & { name: string })[] = [];
  for (const b of blocks) {
    if (skip.has(b.id)) continue;
    const value = (name: string) => (results.get(b.id) ?? []).find((r) => r.name === name)?.value;
    const pub = value("public_address");
    const url = value("url");
    if (b.kind === VM_KIND && isSimulated(results, b.id)) continue;
    if (b.kind === VM_KIND && pub) {
      out.push(
        web.has(b.id)
          ? { kind: "website", id: b.id, name: b.name, label: "", url: `http://${pub}` }
          : { kind: "server", id: b.id, name: b.name, label: "", address: pub },
      );
    } else if (url && linkFor(url)) {
      out.push({ kind: "api", id: b.id, name: b.name, label: "", url });
    }
  }
  const count = (k: LeadLine["kind"]) => out.filter((l) => l.kind === k).length;
  return out.map(({ name, ...l }) => ({ ...l, label: count(l.kind) > 1 ? `${BASE_LABEL[l.kind]} (${name})` : BASE_LABEL[l.kind] }));
}

// sampleNote is the one line that says a function still runs Clutter's
// sample code, or null when none does. Under an API's line it needs no name.
export function sampleNote(results: Map<string, BlockResult[]>, blocks: BlockRef[], hasApiLine: boolean, excluded: string[] = []): string | null {
  const skip = new Set(excluded);
  const names = blocks
    .filter((b) => !skip.has(b.id) && (results.get(b.id) ?? []).some((r) => r.name === CODE && r.value === SAMPLE))
    .map((b) => b.name);
  if (names.length === 0) return null;
  if (names.length === 1 && hasApiLine) return "Answers with a sample until you upload your code.";
  return `${joinAnd(names)} ${names.length === 1 ? "answers" : "answer"} with a sample until you upload your code.`;
}

// simulatedNote is the one line for the VMs a cloud emulator only recorded,
// per cloud in the order given ("VMs on local AWS are simulated; they do not
// boot."), or [] when none is. cloudOf names the cloud a block ran on.
export function simulatedNote(
  results: Map<string, BlockResult[]>,
  blocks: BlockRef[],
  cloudOf: (id: string) => string,
  excluded: string[] = [],
): string[] {
  const skip = new Set(excluded);
  const clouds = blocks.filter((b) => !skip.has(b.id) && b.kind === VM_KIND && isSimulated(results, b.id)).map((b) => cloudOf(b.id));
  return [...new Set(clouds)].map((c) => `VMs on local ${c} are simulated; they do not boot.`);
}

// A document's block and line, as much as servesWeb reads.
export type DocBlock = { id: string; kind: string; fields: Record<string, unknown> };
export type DocLine = { source: string; target: string };

// A firewall entry that lets TCP port 80 in: "tcp/80" or a range holding it
// (connectors.ParseIngress owns the grammar).
const allowsHttp = (ingress: unknown) =>
  typeof ingress === "string" &&
  ingress.split(",").some((item) => {
    const m = item.trim().match(/^tcp\/(\d{1,5})(?:-(\d{1,5}))?$/);
    if (!m) return false;
    const from = Number(m[1]);
    const to = m[2] ? Number(m[2]) : from;
    return from <= 80 && 80 <= to;
  });

// A startup script with something in it: the only way the diagram says what
// a server runs.
const hasStartupScript = (fields: Record<string, unknown>) => typeof fields.startup_script === "string" && fields.startup_script.trim() !== "";

// servesWeb is the servers the diagram makes a website: public, with a
// firewall line that lets the web in on port 80, and a startup script to
// start what serves it. A server without a script runs nothing on port 80,
// so it is "Your server", not a website.
export function servesWeb(blocks: DocBlock[], lines: DocLine[]): Set<string> {
  const byId = new Map(blocks.map((b) => [b.id, b]));
  const out = new Set<string>();
  for (const l of lines) {
    const fw = byId.get(l.source);
    const vm = byId.get(l.target);
    if (fw?.kind === "firewall" && vm?.kind === VM_KIND && vm.fields.public === true && hasStartupScript(vm.fields) && allowsHttp(fw.fields.ingress)) {
      out.add(vm.id);
    }
  }
  return out;
}
