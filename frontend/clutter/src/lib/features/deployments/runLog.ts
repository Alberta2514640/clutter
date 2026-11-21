// runResult reads a run's log for the one line a person wants first: what the
// run did ("Apply complete: 24 added") or why it failed. It reads Terraform's
// own summary lines and B's; a log with none gives null, and the log itself
// stays the full record.
//
// One summary per kind: Terraform prints a plan inside every apply and
// destroy, and B repeats the plan's counts, so the last line of each kind
// stands, and a plan is dropped once the run applied or destroyed. B's own
// apply line counts blocks, not resources, so Terraform's count wins over it.
//
// A run split across clouds has a part per cloud, each under a header ("==
// Lab (Mock cloud): apply =="). B ends a split plan with its own total, so the
// plan reads as above. Terraform's apply and destroy lines stand once per part
// and are added up across the parts. B labels its apply line for each cloud
// ("Lab (Mock cloud): apply complete, 4 blocks built or changed."); when a cloud ran no
// Terraform, those lines stand for the apply instead, one per cloud.

// Relative imports: Vitest runs this file without the "@/" alias.
import { plural } from "../../text";
import { builtBlocks } from "./results";
import type { RunStatus } from "./types";

export type RunResult = { tone: "ok" | "error"; text: string };

type Kind = "plan" | "apply" | "destroy" | "none";
// from: Terraform's own line, or B's block count (one cloud's, when labelled).
// n: the counts of Terraform's apply or destroy line, for adding up parts.
// destroys: a plan's count to destroy.
type Summary = { kind: Kind; text: string; from: "terraform" | "blocks"; cloud?: string; n?: number[]; destroys?: number };

const counts = (n: number[], words: string[]) => {
  const shown = n.map((v, i) => (v !== 0 ? `${v} ${words[i]}` : "")).filter(Boolean);
  return shown.length > 0 ? shown.join(", ") : "no changes";
};

const APPLY = ["added", "changed", "destroyed"];
const DESTROY = ["destroyed"];
const done = (kind: Kind, n: number[]) =>
  kind === "apply" ? `Apply complete: ${counts(n, APPLY)}` : `Destroy complete: ${counts(n, DESTROY)}`;

// What B's apply line counts: the blocks the run built or changed, or none.
const UP_TO_DATE = "everything is up to date";
const blocks = (n: string) => (n === UP_TO_DATE ? UP_TO_DATE : `${n} ${n === "1" ? "block" : "blocks"} built or changed`);
const COUNT = `(?:(\\d+) blocks? built or changed|${UP_TO_DATE})\\.?$`;
const BLOCKS = new RegExp(`^Apply complete: ${COUNT}`);
const BLOCKS_SPLIT = new RegExp(`^(.+): apply complete, ${COUNT}`);

// The header B writes before each cloud's part of a split run.
const PART = /^== .+: (plan|apply|destroy|roll back) ==$/;

function summary(line: string): Summary | null {
  let m = line.match(/Apply complete! Resources: (\d+) added, (\d+) changed, (\d+) destroyed/);
  if (m) {
    const n = [+m[1], +m[2], +m[3]];
    return { kind: "apply", from: "terraform", n, text: done("apply", n) };
  }
  m = line.match(/Destroy complete! Resources: (\d+) destroyed/);
  if (m) return { kind: "destroy", from: "terraform", n: [+m[1]], text: done("destroy", [+m[1]]) };
  m = line.match(/Plan: (\d+) to add, (\d+) to change, (\d+) to destroy/);
  if (m) return { kind: "plan", from: "terraform", destroys: +m[3], text: `Plan: ${counts([+m[1], +m[2], +m[3]], ["to add", "to change", "to destroy"])}` };
  if (/No changes\. Your infrastructure matches the configuration\./.test(line)) return { kind: "none", from: "terraform", text: "No changes" };
  // B's lines: "Apply complete: 6 blocks built or changed." or "Apply
  // complete: everything is up to date." and, per cloud of a split run, "Lab
  // (Mock cloud): apply complete, 4 blocks built or changed."
  m = line.match(BLOCKS_SPLIT);
  if (m) return { kind: "apply", from: "blocks", cloud: m[1], text: `${m[1]}: ${blocks(m[2] ?? UP_TO_DATE)}` };
  m = line.match(BLOCKS);
  if (m) return { kind: "apply", from: "blocks", text: `Apply complete: ${blocks(m[1] ?? UP_TO_DATE)}` };
  return null;
}

// Terraform colours its output; the stored log may keep the escape codes.
const logLines = (log: string) => log.replace(/\u001b\[[0-9;]*m/g, "").split("\n").map((l) => l.trim());

// Terraform draws a box around each error ("╷", "│ Error: ...", "╵").
const BOX = /^[╷│╵]\s*/;

// failureLines is what a failed run's log says went wrong, in plain words:
// each error line (B's "Error: ...", Terraform's boxed "│ Error: ...") and
// each problem line B wrote before the run, without the "Error:" and
// "Problem:" words, once each and at most three. A split run's cloud label
// ("Prod (AWS): ") stays in front. The log itself stays the full record.
export function failureLines(log: string): string[] {
  const out: string[] = [];
  for (const raw of logLines(log)) {
    const line = raw.replace(BOX, "");
    const m = line.match(/^(.*?)\b(?:Error|Problem): (.+)$/);
    if (!m) continue;
    const text = `${m[1]}${m[2]}`.trim();
    if (text && !out.includes(text)) out.push(text);
    if (out.length === 3) break;
  }
  return out;
}

// Known cloud errors, each with one plain sentence of advice. The provider's
// own words often hide the cause (Nova's "No valid host was found" reaches
// Terraform as "unexpected state 'ERROR' ... last error: %!s(<nil>)"), so the
// patterns read the whole log of the failed run. The first match wins.
const KNOWN_ERRORS: { pattern: RegExp; advice: string }[] = [
  {
    // Nova's server states are upper case; a Cinder volume's "error" is not.
    pattern: /No valid host was found|to become ready: unexpected state 'ERROR'|unexpected state 'ERROR', wanted target 'ACTIVE'/,
    advice:
      "The cloud could not start the server: it may be out of room for this size, or the image and size do not fit. Try a smaller size, or check the server in your cloud's console.",
  },
  { pattern: /resizing online is disabled/i, advice: "This cloud cannot grow a disk while it is attached." },
  { pattern: /Duplicate items in the list/, advice: "The cloud refused a firewall change on a running server." },
  { pattern: /UnauthorizedOperation|AccessDenied/, advice: "Clutter's role in your account is not allowed to do this. Update the account link's stack." },
  { pattern: /quota ?exceeded|exceeded (?:your |the )?quota|LimitExceeded/i, advice: "Your cloud account has reached a limit." },
  { pattern: /timeout while waiting[^\n]*delet|to become deleted/i, advice: "The cloud took too long to remove something another resource still uses." },
];

// failureAdvice is the advice for the first known error in a failed run's
// log, or null when the log holds none.
export function failureAdvice(log: string): string | null {
  const text = logLines(log).join("\n");
  return KNOWN_ERRORS.find((k) => k.pattern.test(text))?.advice ?? null;
}

// Terraform's line for each resource a plan removes and builds again:
// "# aws_x.y must be replaced", "# aws_x.y is tainted, so must be
// replaced", "# aws_x.y will be replaced, as requested".
const REPLACED = /^# \S+ (?:(?:is tainted, so )?must be replaced|will be replaced\b)/;

// replacedNote says how many of a plan's destroys are replacements, which
// Terraform counts once as an add and once as a destroy: " (4 of them are
// replaced: removed and built again)". "" when none is, or the log does not
// say.
function replacedNote(lines: string[], destroys: number): string {
  const replaced = Math.min(lines.filter((l) => REPLACED.test(l)).length, destroys);
  if (replaced === 0) return "";
  if (replaced === destroys) return ` (${destroys === 1 ? "it is" : `all ${destroys} are`} replaced: removed and built again)`;
  return ` (${replaced} of them ${replaced === 1 ? "is" : "are"} replaced: removed and built again)`;
}

export function runResult(log: string, status: RunStatus): RunResult | null {
  if (status === "failed") {
    const [first] = failureLines(log);
    return first ? { tone: "error", text: first } : null;
  }
  if (status !== "completed") return null;
  const lines = logLines(log);
  const { found } = summaries(lines);
  const text = (s: Summary) => (s.kind === "plan" && s.destroys ? `${s.text}${replacedNote(lines, s.destroys)}` : s.text);
  return found.length > 0 ? { tone: "ok", text: found.map(text).join("; ") } : null;
}

// blocksChanged adds up B's own apply lines (one per cloud of a split run):
// the blocks the run built or changed, or null for a log with none.
function blocksChanged(lines: string[]): number | null {
  let total: number | null = null;
  for (const line of lines) {
    const m = line.match(BLOCKS_SPLIT) ?? line.match(BLOCKS);
    if (m) total = (total ?? 0) + Number(m[m.length - 1] ?? 0);
  }
  return total;
}

// B's destroy line when the connector names what it removed: "Destroy
// complete: 3 blocks removed." A Terraform connector's destroy has none.
const REMOVED = /^Destroy complete: (\d+) blocks? removed\.?$/;

// runHeadline is the first line a person reads after a run, counted from the
// run's own log, never from the diagram as it is now. A finished deploy says
// the blocks it built (its "Built:" lines) and the cloud resources behind
// them ("Built 4 blocks (18 cloud resources)"); a finished destroy what it
// removed ("Destroyed 3 blocks"); a failed run its first error in plain
// words; anything else runResult's line.
export function runHeadline(log: string, status: RunStatus, action: string): RunResult | null {
  if (status !== "completed") return runResult(log, status);
  const lines = logLines(log);
  const { totals } = summaries(lines);
  if (action === "destroy") {
    const removed = lines.map((l) => l.match(REMOVED)).find(Boolean);
    const resources = totals.get("destroy")?.[0];
    const behind = resources ? ` (${plural(resources, "cloud resource", "cloud resources")})` : "";
    if (removed) return { tone: "ok", text: `Destroyed ${plural(Number(removed[1]), "block", "blocks")}${behind}` };
    if (resources === 0) return { tone: "ok", text: "Nothing was left to destroy." };
    if (resources) return { tone: "ok", text: `Destroyed ${plural(resources, "cloud resource", "cloud resources")}` };
    return { tone: "ok", text: "Destroyed everything this deployment built." };
  }
  if (action !== "apply") return runResult(log, status);
  // A run from before B named the blocks it built has no count of its own.
  const blocks = builtBlocks(log, [])?.length;
  if (blocks === undefined) return runResult(log, status);
  const built = `Built ${plural(blocks, "block", "blocks")}`;
  const n = totals.get("apply");
  if (!n) {
    // No Terraform (the mock): B's own count says what changed.
    const changed = blocksChanged(lines);
    if (changed === 0) return { tone: "ok", text: `${built}. Nothing needed to change.` };
    if (changed !== null && changed < blocks) return { tone: "ok", text: `${built} (${changed} new or changed)` };
    return { tone: "ok", text: built };
  }
  const [added, changed, destroyed] = n;
  if (added + changed + destroyed === 0) return { tone: "ok", text: `${built}. Nothing needed to change.` };
  if (changed === 0 && destroyed === 0) return { tone: "ok", text: `${built} (${plural(added, "cloud resource", "cloud resources")})` };
  const parts = [added > 0 && `${plural(added, "cloud resource", "cloud resources")} added`, changed > 0 && `${changed} changed`, destroyed > 0 && `${destroyed} removed`];
  return { tone: "ok", text: `${built} (${parts.filter(Boolean).join(", ")})` };
}

function summaries(lines: string[]): { found: Summary[]; totals: Map<Kind, number[]> } {
  // A Map keeps each key where it was first seen; set replaces the value.
  const byKey = new Map<string, Summary>();
  let part = 0;
  for (const line of lines) {
    if (PART.test(line)) {
      part++;
      continue;
    }
    const s = summary(line);
    if (!s) continue;
    const scope = s.cloud ?? (s.kind === "apply" || s.kind === "destroy" ? String(part) : "");
    const key = `${scope}\u0000${s.kind}`;
    const prev = byKey.get(key);
    if (prev?.from === "terraform" && s.from === "blocks") continue;
    byKey.set(key, s);
  }
  let found = [...byKey.values()];
  const labelled = found.filter((s) => s.cloud).length;
  if (labelled > 0) {
    const counted = found.filter((s) => s.kind === "apply" && s.n).length;
    found = counted >= labelled ? found.filter((s) => !s.cloud) : found.filter((s) => s.cloud || s.kind !== "apply");
  }
  // Add up Terraform's counts of the same kind across the parts.
  const totals = new Map<Kind, number[]>();
  for (const s of found) if (s.n) totals.set(s.kind, (totals.get(s.kind) ?? s.n.map(() => 0)).map((v, i) => v + s.n![i]));
  found = found.filter((s, i) => !s.n || found.findIndex((t) => t.n && t.kind === s.kind) === i);
  found = found.map((s) => (s.n ? { ...s, text: done(s.kind, totals.get(s.kind)!) } : s));
  if (found.some((s) => s.kind === "apply" || s.kind === "destroy")) found = found.filter((s) => s.kind === "apply" || s.kind === "destroy");
  return { found, totals };
}
