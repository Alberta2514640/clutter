"use client";

import { adapterErrorMessage } from "@/lib/adapterApi";
import { blockLook } from "@/lib/features/catalog/display";
import type { CatalogEntry } from "@/lib/features/catalog/types";
import type { BlockRef } from "@/lib/features/deployments/compatibility";
import { useRunLogs } from "@/lib/features/deployments/hooks";
import { leadLines, parseResults, resultRows, sampleNote, simulatedNote, type LeadLine } from "@/lib/features/deployments/results";
import type { DeploymentListItem, Run } from "@/lib/features/deployments/types";
import { Check, ChevronDown, ChevronRight, Copy, ExternalLink, Loader2 } from "lucide-react";
import { useEffect, useState } from "react";
import KindIcon from "./KindIcon";

type Props = {
  token: string | null;
  organizationId: string;
  item: DeploymentListItem;
  // The deployment's last successful apply, which built what is listed.
  run: Run;
  catalog: CatalogEntry[];
  blocks: BlockRef[];
  blockTitle: (kind: string) => string;
  // The servers the diagram makes a website (results.servesWeb).
  webServers: ReadonlySet<string>;
};

// WhatWasBuilt shows what the last successful apply built, and says so. It
// leads with one plain line per result a person can use: the website's
// address to open, the API's address to copy or open, and whether the API
// still answers with Clutter's sample. A VM a cloud emulator only recorded
// gets no line to open, and one sentence says why. Each block, what it
// became on its cloud and every value the apply returned sit behind Details.
// It reads them from the run's log, where B writes them
// (deployments/results.ts).
export default function WhatWasBuilt({ token, organizationId, item, run, catalog, blocks, blockTitle, webServers }: Props) {
  const logQ = useRunLogs(token, organizationId, run.id, run.status);
  const [detailsOpen, setDetailsOpen] = useState(false);
  const manifestOf = (connector: string) => catalog.find((e) => e.manifest.connector === connector)?.manifest ?? null;
  // A block placed on another cloud became that cloud's service.
  const cloudOf = (id: string) => item.placements?.find((p) => (p.node_ids ?? []).includes(id))?.connector ?? item.connector;
  const became = (b: BlockRef) => {
    const m = manifestOf(cloudOf(b.id));
    return m ? blockLook(m, b.kind, blockTitle(b.kind)).name : blockTitle(b.kind);
  };

  if (logQ.isLoading) {
    return (
      <p role="status" className="flex items-center gap-1.5 text-xs text-slate-400">
        <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden />
        Reading what was built…
      </p>
    );
  }
  if (logQ.isError) return <p className="text-xs text-red-300">Could not read what was built: {adapterErrorMessage(logQ.error)}</p>;

  const excluded = item.exclusions ?? [];
  const results = parseResults(logQ.data ?? "");
  const lead = leadLines(results, blocks, webServers, excluded);
  const sample = sampleNote(results, blocks, lead.some((l) => l.kind === "api"), excluded);
  const rows = resultRows(results, blocks, became, excluded);
  const simulated = simulatedNote(results, blocks, (id) => manifestOf(cloudOf(id))?.display?.short_name || cloudOf(id), excluded);
  // With nothing to open, the details are the answer, so they show at once.
  const showDetails = detailsOpen || (lead.length === 0 && !sample);
  const when = run.finished_at ? new Date(run.finished_at).toLocaleString() : null;

  return (
    <section aria-label="What was built" data-testid="what-was-built" className="space-y-2 rounded-lg border border-emerald-900/60 bg-emerald-950/20 p-2.5">
      <p data-testid="built-by-last-deploy" className="text-[11px] text-slate-400">
        From the last deploy{when ? `, ${when}` : ""}
      </p>
      {simulated.map((s) => (
        <p key={s} data-testid="simulated-note" className="text-xs text-amber-200">
          {s}
        </p>
      ))}
      {lead.length > 0 && (
        <ul className="space-y-1.5" aria-label="Ready to use">
          {lead.map((l) => (
            <li key={l.id} data-testid="result-line" className="space-y-0.5">
              <ResultLine line={l} />
              {l.kind === "api" && sample && <p className="text-[11px] text-slate-400">{sample}</p>}
            </li>
          ))}
        </ul>
      )}
      {sample && !lead.some((l) => l.kind === "api") && <p className="text-[11px] text-slate-400">{sample}</p>}
      <div>
        {(lead.length > 0 || sample) && (
          <button
            type="button"
            aria-expanded={detailsOpen}
            onClick={() => setDetailsOpen((o) => !o)}
            className="flex items-center gap-1 text-[11px] text-slate-400 hover:text-white">
            {detailsOpen ? <ChevronDown className="h-3 w-3" aria-hidden /> : <ChevronRight className="h-3 w-3" aria-hidden />}
            Details
          </button>
        )}
        {showDetails && (
          <ul className="mt-1.5 space-y-1.5" aria-label="Each block">
            {rows.map((r) => (
              <li key={r.id} data-testid="built-block" className="text-xs">
                <div className="flex min-w-0 items-center gap-1.5">
                  <KindIcon kind={r.kind} className="h-3.5 w-3.5 shrink-0 text-teal-300" />
                  <span className="truncate font-medium text-slate-100">{r.name}</span>
                  <span className="truncate text-slate-500">{r.became}</span>
                </div>
                {r.items.map((i) => (
                  <div key={i.label} className="flex min-w-0 items-baseline gap-2 pl-5 text-[11px]">
                    <span className="shrink-0 text-slate-500">{i.label}</span>
                    <span data-testid="result-value" className="min-w-0 select-all break-all font-mono text-slate-200">
                      {i.value}
                    </span>
                  </div>
                ))}
              </li>
            ))}
          </ul>
        )}
      </div>
    </section>
  );
}

const linkClass =
  "inline-flex shrink-0 items-center gap-1 rounded border border-slate-700 px-1.5 py-0.5 text-[11px] text-slate-200 hover:border-teal-500 hover:text-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-teal-300";

// One lead line: its label, the full address (never cut: it wraps, and its
// title holds it too), and its actions.
function ResultLine({ line }: { line: LeadLine }) {
  const value = line.kind === "server" ? line.address : line.url;
  return (
    <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs">
      <span className="shrink-0 font-medium text-emerald-200">{line.label}:</span>
      <span title={value} className="min-w-0 break-all font-mono text-slate-100">
        {value}
      </span>
      <span className="flex shrink-0 items-center gap-1">
        {line.kind !== "website" && <CopyButton value={value} label={line.label} />}
        {line.kind !== "server" && (
          <a href={line.url} target="_blank" rel="noopener noreferrer" data-testid="result-link" aria-label={`Open ${line.label.toLowerCase()}`} className={linkClass}>
            <ExternalLink className="h-3 w-3" aria-hidden />
            Open
          </a>
        )}
      </span>
    </div>
  );
}

// CopyButton puts value on the clipboard and says so for a moment. A browser
// that refuses the clipboard gets a plain sentence; the value stays on screen
// to select by hand.
function CopyButton({ value, label }: { value: string; label: string }) {
  const [state, setState] = useState<"idle" | "copied" | "failed">("idle");
  useEffect(() => {
    if (state === "idle") return;
    const t = window.setTimeout(() => setState("idle"), 2000);
    return () => window.clearTimeout(t);
  }, [state]);
  return (
    <button
      type="button"
      aria-label={`Copy ${label.toLowerCase()}`}
      onClick={() => {
        if (!navigator.clipboard) return setState("failed");
        navigator.clipboard.writeText(value).then(
          () => setState("copied"),
          () => setState("failed"),
        );
      }}
      className={linkClass}>
      {state === "copied" ? <Check className="h-3 w-3" aria-hidden /> : <Copy className="h-3 w-3" aria-hidden />}
      <span aria-live="polite">{state === "copied" ? "Copied" : state === "failed" ? "Select it to copy" : "Copy"}</span>
    </button>
  );
}
