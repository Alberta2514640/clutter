// runResult reads a run's log for the one line a person wants first: what the
// run did ("Apply complete: 24 added") or why it failed. It reads Terraform's
// own summary lines and B's; a log with none gives null, and the log itself
// stays the full record.
//
// One summary per kind: Terraform prints a plan inside every apply and
// destroy, and B repeats the plan's counts, so the last line of each kind
// stands, and a plan is dropped once the run applied or destroyed. B's own
// apply line counts blocks, not resources, so Terraform's count wins over it.
// A run split across clouds labels B's line for each cloud ("Lab (Mock
// cloud): apply complete, 4 blocks built."); those lines stand for the apply,
// one per cloud.

// Relative imports: Vitest runs this file without the "@/" alias.
import type { RunStatus } from "./types";

export type RunResult = { tone: "ok" | "error"; text: string };

type Kind = "plan" | "apply" | "destroy" | "none";
// from: Terraform's own line, or B's block count (one cloud's, when labelled).
type Summary = { kind: Kind; text: string; from: "terraform" | "blocks"; cloud?: string };

const counts = (pairs: [string, string][]) => {
  const shown = pairs.filter(([n]) => n !== "0").map(([n, word]) => `${n} ${word}`);
  return shown.length > 0 ? shown.join(", ") : "no changes";
};

const blocks = (n: string) => `${n} ${n === "1" ? "block" : "blocks"} built`;

function summary(line: string): Summary | null {
  let m = line.match(/Apply complete! Resources: (\d+) added, (\d+) changed, (\d+) destroyed/);
  if (m) return { kind: "apply", from: "terraform", text: `Apply complete: ${counts([[m[1], "added"], [m[2], "changed"], [m[3], "destroyed"]])}` };
  m = line.match(/Destroy complete! Resources: (\d+) destroyed/);
  if (m) return { kind: "destroy", from: "terraform", text: `Destroy complete: ${counts([[m[1], "destroyed"]])}` };
  m = line.match(/Plan: (\d+) to add, (\d+) to change, (\d+) to destroy/);
  if (m) return { kind: "plan", from: "terraform", text: `Plan: ${counts([[m[1], "to add"], [m[2], "to change"], [m[3], "to destroy"]])}` };
  if (/No changes\. Your infrastructure matches the configuration\./.test(line)) return { kind: "none", from: "terraform", text: "No changes" };
  // B's lines: "Apply complete: 6 blocks built." and, per cloud of a split
  // run, "Lab (Mock cloud): apply complete, 4 blocks built."
  m = line.match(/^(.+): apply complete, (\d+) blocks? built\.?$/);
  if (m) return { kind: "apply", from: "blocks", cloud: m[1], text: `${m[1]}: ${blocks(m[2])}` };
  m = line.match(/^Apply complete: (\d+) blocks? built\.?$/);
  if (m) return { kind: "apply", from: "blocks", text: `Apply complete: ${blocks(m[1])}` };
  return null;
}

export function runResult(log: string, status: RunStatus): RunResult | null {
  // Terraform colours its output; the stored log may keep the escape codes.
  const lines = log.replace(/\u001b\[[0-9;]*m/g, "").split("\n");
  if (status === "failed") {
    const error = lines.map((l) => l.trim()).find((l) => /\bError:/.test(l));
    return error ? { tone: "error", text: error.slice(error.indexOf("Error:")) } : null;
  }
  if (status !== "completed") return null;

  // A Map keeps each key where it was first seen; set replaces the value.
  const byKey = new Map<string, Summary>();
  for (const s of lines.map((l) => summary(l.trim())).filter((s): s is Summary => !!s)) {
    const key = s.cloud ? `${s.cloud}\u0000${s.kind}` : s.kind;
    const prev = byKey.get(key);
    if (prev?.from === "terraform" && s.from === "blocks") continue;
    byKey.set(key, s);
  }
  let found = [...byKey.values()];
  if (found.some((s) => s.cloud)) found = found.filter((s) => s.cloud || s.kind !== "apply");
  if (found.some((s) => s.kind === "apply" || s.kind === "destroy")) found = found.filter((s) => s.kind === "apply" || s.kind === "destroy");
  return found.length > 0 ? { tone: "ok", text: found.map((s) => s.text).join("; ") } : null;
}
