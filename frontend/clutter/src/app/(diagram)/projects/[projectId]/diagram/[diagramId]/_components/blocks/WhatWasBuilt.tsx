"use client";

import { adapterErrorMessage } from "@/lib/adapterApi";
import { blockLook, kindIconClass } from "@/lib/features/catalog/display";
import { entryFor } from "@/lib/features/catalog/forms";
import type { CatalogEntry } from "@/lib/features/catalog/types";
import type { BlockRef } from "@/lib/features/deployments/compatibility";
import { useRunLogs } from "@/lib/features/deployments/hooks";
import { builtBlocks, leadLines, parseResults, resultRows, sampleNote, simulatedNote, type LeadLine } from "@/lib/features/deployments/results";
import { diagramNotDeployed, type DeploymentListItem, type Run } from "@/lib/features/deployments/types";
import CopyButton, { SMALL_ACTION } from "@/components/common/CopyButton";
import { ChevronDown, ChevronRight, ExternalLink, Loader2 } from "lucide-react";
import { useState } from "react";
import KindIcon from "./KindIcon";

type Props = {
  token: string | null;
  organizationId: string;
  item: DeploymentListItem;
  // The deployment's last successful apply, which built what is listed.
  run: Run;
  catalog: CatalogEntry[];
  // The saved diagram's blocks: the order to list built blocks in, and what
  // to list for a run from before B named the blocks it built.
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
// (deployments/results.ts): the blocks and their names as that apply built
// them, so a rename saved since does not show here until it is deployed.
export default function WhatWasBuilt({ token, organizationId, item, run, catalog, blocks: saved, blockTitle, webServers }: Props) {
  const logQ = useRunLogs(token, organizationId, run);
  const [detailsOpen, setDetailsOpen] = useState(false);
  const manifestOf = (connector: string) => entryFor(catalog, connector)?.manifest ?? null;
  // A block placed on another cloud became that cloud's service.
  const cloudOf = (id: string) => item.placements?.find((p) => (p.node_ids ?? []).includes(id))?.connector ?? item.connector;
  const became = (b: BlockRef) => {
    const m = manifestOf(cloudOf(b.id));
    return m ? blockLook(m, b.kind, blockTitle(b.kind)).name : blockTitle(b.kind);
  };

  if (logQ.isLoading) {
    return (
      <p role="status" className="text-muted-foreground flex items-center gap-1.5 text-xs">
        <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden />
        Reading what was built…
      </p>
    );
  }
  if (logQ.isError) return <p className="text-danger text-xs">Could not read what was built: {adapterErrorMessage(logQ.error)}</p>;

  const log = logQ.data ?? "";
  // The blocks the apply built, as it named them. A run from before B wrote
  // them lists the saved blocks, less those left out, and says so once the
  // diagram has changed since.
  const asBuilt = builtBlocks(log, saved);
  const blocks = asBuilt ?? saved;
  const excluded = asBuilt ? [] : (item.exclusions ?? []);
  // Names change only with the diagram, not with the deployment's settings.
  const namesAsSaved = !asBuilt && diagramNotDeployed(item);
  const results = parseResults(log);
  const lead = leadLines(results, blocks, webServers, excluded);
  const sample = sampleNote(results, blocks, lead.some((l) => l.kind === "api"), excluded);
  const rows = resultRows(results, blocks, became, excluded);
  const simulated = simulatedNote(results, blocks, (id) => manifestOf(cloudOf(id))?.display?.short_name || cloudOf(id), excluded);
  // With nothing to open, the details are the answer, so they show at once.
  const showDetails = detailsOpen || (lead.length === 0 && !sample);
  const when = run.finished_at ? new Date(run.finished_at).toLocaleString() : null;

  return (
    <section aria-label="What was built" data-testid="what-was-built" className="border-success/25 bg-success/5 space-y-2 rounded-lg border p-3">
      <p className="text-sm font-semibold">What was built</p>
      <p data-testid="built-by-last-deploy" className="text-muted-foreground text-[11px]">
        From the last deploy{when ? `, ${when}` : ""}
      </p>
      {namesAsSaved && (
        <p data-testid="names-as-saved" className="text-muted-foreground text-[11px]">
          The diagram changed since this deploy. Names show as saved now.
        </p>
      )}
      {simulated.map((s) => (
        <p key={s} data-testid="simulated-note" className="text-warning text-xs">
          {s}
        </p>
      ))}
      {lead.length > 0 && (
        <ul className="space-y-1.5" aria-label="Ready to use">
          {lead.map((l) => (
            <li key={l.id} data-testid="result-line" className="space-y-0.5">
              <ResultLine line={l} />
              {l.kind === "api" && sample && <p className="text-muted-foreground text-[11px]">{sample}</p>}
            </li>
          ))}
        </ul>
      )}
      {sample && !lead.some((l) => l.kind === "api") && <p className="text-muted-foreground text-[11px]">{sample}</p>}
      <div>
        {(lead.length > 0 || sample) && (
          <button
            type="button"
            aria-expanded={detailsOpen}
            onClick={() => setDetailsOpen((o) => !o)}
            className="text-muted-foreground hover:text-foreground flex min-h-6 items-center gap-1 text-xs">
            {detailsOpen ? <ChevronDown className="size-3" aria-hidden /> : <ChevronRight className="size-3" aria-hidden />}
            Details
          </button>
        )}
        {showDetails && (
          <ul className="mt-1.5 space-y-1.5" aria-label="Each block">
            {rows.map((r) => (
              <li key={r.id} data-testid="built-block" className="text-xs">
                <div className="flex min-w-0 items-center gap-1.5">
                  <KindIcon kind={r.kind} className={`${kindIconClass(r.kind)} size-3.5 shrink-0`} />
                  <span className="truncate font-medium">{r.name}</span>
                  <span className="text-muted-foreground truncate">{r.became}</span>
                </div>
                {r.items.map((i) => (
                  <div key={i.label} className="flex min-w-0 items-baseline gap-2 pl-5 text-[11px]">
                    <span className="text-muted-foreground shrink-0">{i.label}</span>
                    <span data-testid="result-value" className="min-w-0 font-mono break-all select-all">
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

// One lead line: its label, the full address (never cut: it wraps, and its
// title holds it too), and its actions.
function ResultLine({ line }: { line: LeadLine }) {
  const value = line.kind === "server" ? line.address : line.url;
  return (
    <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs">
      <span className="text-success shrink-0 font-medium">{line.label}:</span>
      <span title={value} className="min-w-0 font-mono break-all">
        {value}
      </span>
      <span className="flex shrink-0 items-center gap-1">
        {line.kind !== "website" && <CopyButton value={value} label={line.label.toLowerCase()} />}
        {line.kind !== "server" && (
          <a href={line.url} target="_blank" rel="noopener noreferrer" data-testid="result-link" aria-label={`Open ${line.label.toLowerCase()}`} className={SMALL_ACTION}>
            <ExternalLink className="size-3" aria-hidden />
            Open
          </a>
        )}
      </span>
    </div>
  );
}
