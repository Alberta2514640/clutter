"use client";

import type { DiagramSummary } from "@/lib/features/diagram/summary";
import { ChevronRight } from "lucide-react";
import { useState } from "react";
import KindIcon from "./KindIcon";

// The one-line summary at the bottom of the canvas ("Builds 2 servers, 1
// network"). It opens to a short list of the blocks, grouped by kind.
export default function SummaryPill({ summary }: { summary: DiagramSummary }) {
  const [open, setOpen] = useState(false);
  return (
    <div className="absolute bottom-3 left-14 z-10 max-w-[calc(100%-5rem)]">
      {open && summary.groups.length > 0 && (
        <ul data-testid="summary-list" className="mb-2 space-y-1.5 rounded-xl border border-slate-800 bg-slate-900/95 px-3 py-2.5 text-xs shadow-xl">
          {summary.groups.map((g) => (
            <li key={g.kind} className="flex items-center gap-2">
              <KindIcon kind={g.kind} className="h-3.5 w-3.5 shrink-0 text-teal-300" />
              <span className="shrink-0 text-slate-200">{g.label}</span>
              <span className="truncate text-slate-500">{g.names.join(", ")}</span>
            </li>
          ))}
        </ul>
      )}
      <button
        type="button"
        data-testid="summary-pill"
        aria-expanded={open}
        disabled={summary.groups.length === 0}
        onClick={() => setOpen((o) => !o)}
        className="inline-flex max-w-full items-center gap-1 rounded-full border border-slate-800 bg-slate-900/95 px-3.5 py-1.5 text-xs text-slate-300 shadow-lg hover:border-slate-700 hover:text-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-teal-300 disabled:cursor-default disabled:hover:border-slate-800 disabled:hover:text-slate-300">
        <span className="truncate">{summary.line}</span>
        {summary.groups.length > 0 && <ChevronRight className={`h-3.5 w-3.5 shrink-0 transition-transform ${open ? "-rotate-90" : ""}`} aria-hidden />}
      </button>
    </div>
  );
}
