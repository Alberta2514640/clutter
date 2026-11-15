"use client";

import { adapterErrorMessage } from "@/lib/adapterApi";
import { blockLook } from "@/lib/features/catalog/display";
import type { CatalogEntry } from "@/lib/features/catalog/types";
import type { BlockRef } from "@/lib/features/deployments/compatibility";
import { useRunLogs } from "@/lib/features/deployments/hooks";
import { parseResults, resultRows } from "@/lib/features/deployments/results";
import type { DeploymentListItem, Run } from "@/lib/features/deployments/types";
import { ExternalLink, Loader2 } from "lucide-react";
import KindIcon from "./KindIcon";

type Props = {
  token: string | null;
  organizationId: string;
  item: DeploymentListItem;
  // The deployment's successful apply.
  run: Run;
  catalog: CatalogEntry[];
  blocks: BlockRef[];
  blockTitle: (kind: string) => string;
};

// WhatWasBuilt lists, after a successful apply, each block, what it became on
// its cloud, and what the apply returned for it: a server's addresses, an HTTP
// API's address as a link, a bucket's and a table's name. It reads them from
// the run's log, where B writes them (deployments/results.ts).
export default function WhatWasBuilt({ token, organizationId, item, run, catalog, blocks, blockTitle }: Props) {
  const logQ = useRunLogs(token, organizationId, run.id, run.status);
  const manifestOf = (connector: string) => catalog.find((e) => e.manifest.connector === connector)?.manifest ?? null;
  // A block placed on another cloud became that cloud's service.
  const cloudOf = (id: string) => item.placements?.find((p) => (p.node_ids ?? []).includes(id))?.connector ?? item.connector;
  const became = (b: BlockRef) => {
    const m = manifestOf(cloudOf(b.id));
    return m ? blockLook(m, b.kind, blockTitle(b.kind)).name : blockTitle(b.kind);
  };

  return (
    <section aria-label="What was built" data-testid="what-was-built" className="space-y-2 rounded-lg border border-emerald-900/60 bg-emerald-950/20 p-2.5">
      <h4 className="text-xs font-semibold text-emerald-200">What was built</h4>
      {logQ.isLoading ? (
        <p role="status" className="flex items-center gap-1.5 text-xs text-slate-400">
          <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden />
          Reading the results…
        </p>
      ) : logQ.isError ? (
        <p className="text-xs text-red-300">Could not read the results: {adapterErrorMessage(logQ.error)}</p>
      ) : (
        <ul className="space-y-1.5">
          {resultRows(parseResults(logQ.data ?? ""), blocks, became, item.exclusions ?? []).map((r) => (
            <li key={r.id} data-testid="built-block" className="text-xs">
              <div className="flex min-w-0 items-center gap-1.5">
                <KindIcon kind={r.kind} className="h-3.5 w-3.5 shrink-0 text-teal-300" />
                <span className="truncate font-medium text-slate-100">{r.name}</span>
                <span className="shrink-0 text-slate-500">→</span>
                <span className="truncate text-slate-400">{r.became}</span>
              </div>
              {r.items.map((i) => (
                <div key={i.label} className="flex min-w-0 items-center gap-2 pl-5 text-[11px]">
                  <span className="shrink-0 text-slate-500">{i.label}</span>
                  {i.href ? (
                    <a
                      href={i.href}
                      target="_blank"
                      rel="noopener noreferrer"
                      data-testid="result-link"
                      className="inline-flex min-w-0 items-center gap-1 truncate font-mono text-teal-300 hover:underline">
                      <span className="truncate">{i.value}</span>
                      <ExternalLink className="h-3 w-3 shrink-0" aria-hidden />
                    </a>
                  ) : (
                    <span data-testid="result-value" className="truncate select-all font-mono text-slate-200">
                      {i.value}
                    </span>
                  )}
                </div>
              ))}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
