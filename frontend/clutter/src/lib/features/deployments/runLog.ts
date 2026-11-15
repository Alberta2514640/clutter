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
import type { RunStatus } from "./types";

export type RunResult = { tone: "ok" | "error"; text: string };

type Kind = "plan" | "apply" | "destroy" | "none";
// from: Terraform's own line, or B's block count (one cloud's, when labelled).
// n: the counts of Terraform's apply or destroy line, for adding up parts.
type Summary = { kind: Kind; text: string; from: "terraform" | "blocks"; cloud?: string; n?: number[] };

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
  if (m) return { kind: "plan", from: "terraform", text: `Plan: ${counts([+m[1], +m[2], +m[3]], ["to add", "to change", "to destroy"])}` };
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

export function runResult(log: string, status: RunStatus): RunResult | null {
  const lines = logLines(log);
  if (status === "failed") {
    const error = lines.find((l) => /\bError:/.test(l));
    return error ? { tone: "error", text: error.slice(error.indexOf("Error:")) } : null;
  }
  if (status !== "completed") return null;
  const { found } = summaries(lines);
  return found.length > 0 ? { tone: "ok", text: found.map((s) => s.text).join("; ") } : null;
}

// runHeadline is the first line a person reads after a run: for a finished
// deploy, the blocks it built and the cloud resources behind them ("Built 4
// blocks (18 cloud resources)"); for anything else, runResult's line.
export function runHeadline(log: string, status: RunStatus, action: string, blocks: number): RunResult | null {
  if (action !== "apply" || status !== "completed") return runResult(log, status);
  const n = summaries(logLines(log)).totals.get("apply");
  const built = `Built ${plural(blocks, "block", "blocks")}`;
  if (!n) return { tone: "ok", text: built };
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
