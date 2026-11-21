"use client";

import { kindIconClass } from "@/lib/features/catalog/display";
import type { DiagramSummary } from "@/lib/features/diagram/summary";
import { ChevronRight } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import KindIcon from "./KindIcon";

// The one-line summary at the bottom of the canvas ("Builds 2 servers, 1
// network"). It opens to a short list of the blocks, grouped by kind, and
// closes on Escape or a click outside, as the palette's "How linking works"
// does. The editor places it, in the canvas's bottom-left row.
export default function SummaryPill({ summary }: { summary: DiagramSummary }) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    // Capture: the canvas stops a press on its pane from reaching the window.
    const onPointer = (e: PointerEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    window.addEventListener("keydown", onKey);
    window.addEventListener("pointerdown", onPointer, true);
    return () => {
      window.removeEventListener("keydown", onKey);
      window.removeEventListener("pointerdown", onPointer, true);
    };
  }, [open]);

  return (
    <div ref={ref} className="pointer-events-auto min-w-0">
      {open && summary.groups.length > 0 && (
        <ul data-testid="summary-list" className="surface-raised text-popover-foreground mb-2 space-y-1.5 rounded-xl border px-3 py-2.5 text-xs animate-fade-up">
          {summary.groups.map((g) => (
            <li key={g.kind} className="flex items-center gap-2">
              <KindIcon kind={g.kind} className={`${kindIconClass(g.kind)} size-3.5 shrink-0`} />
              <span className="shrink-0">{g.label}</span>
              <span className="text-muted-foreground truncate">{g.names.join(", ")}</span>
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
        className="surface-raised text-muted-foreground hover:text-foreground focus-visible:ring-ring/50 inline-flex h-8 max-w-full items-center gap-1 rounded-full border px-3.5 text-xs transition-colors outline-none focus-visible:ring-[3px] disabled:cursor-default">
        <span className="truncate">{summary.line}</span>
        {summary.groups.length > 0 && <ChevronRight className={`size-3.5 shrink-0 transition-transform ${open ? "-rotate-90" : ""}`} aria-hidden />}
      </button>
    </div>
  );
}
