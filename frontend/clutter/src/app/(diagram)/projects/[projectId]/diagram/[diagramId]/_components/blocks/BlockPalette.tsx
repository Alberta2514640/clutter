"use client";

import type { PaletteBlock } from "@/lib/features/catalog/forms";
import { NETWORK_KIND } from "@/lib/features/diagram/blocksDocument";
import { Link2, Loader2, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import type { EdgeCard } from "./canvasContext";
import { LinkCardBody } from "./FloatingEdge";
import KindIcon from "./KindIcon";

export const BLOCK_DND_MIME = "application/x-clutter-block";

type Props = {
  blocks: PaletteBlock[];
  loading: boolean;
  error: string | null;
  onAdd: (kind: string) => void;
  // Called with the kind a drag starts with, and null when it ends.
  onDragKind: (kind: string | null) => void;
  // Every line the clouds in play draw, for "How linking works".
  links: EdgeCard[];
};

// The palette lists the blocks the connected clouds can build, read from the
// manifests: each with its icon, its name and one short line on what it is
// for. Drag a block onto the canvas, or click it to add it. Which clouds build
// it is in its tooltip.
export default function BlockPalette({ blocks, loading, error, onAdd, onDragKind, links }: Props) {
  return (
    <aside aria-label="Building blocks" className="flex h-full w-60 shrink-0 flex-col border-r border-slate-800 bg-slate-950">
      <div className="border-b border-slate-800 px-4 py-3">
        <h2 className="text-sm font-semibold text-white">Building blocks</h2>
      </div>
      <div className="flex-1 overflow-y-auto p-3">
        {loading ? (
          <div role="status" className="flex items-center gap-2 text-sm text-slate-400">
            <Loader2 className="h-4 w-4 animate-spin" />
            Loading blocks…
          </div>
        ) : error ? (
          <p role="alert" className="text-sm text-red-400">
            Could not load the blocks: {error}
          </p>
        ) : blocks.length === 0 ? (
          <p className="text-sm text-slate-400">No cloud offers any blocks yet.</p>
        ) : (
          <ul className="space-y-2">
            {blocks.map((b) => {
              const builtBy = `Built by ${b.clouds.map((c) => c.title).join(", ")}`;
              const box = b.kind === NETWORK_KIND;
              return (
                <li key={b.kind}>
                  <button
                    type="button"
                    draggable
                    data-testid={`palette-${b.kind}`}
                    onDragStart={(e) => {
                      e.dataTransfer.setData(BLOCK_DND_MIME, b.kind);
                      e.dataTransfer.effectAllowed = "move";
                      onDragKind(b.kind);
                      // Drag a card shaped like the block on the canvas, not this palette row.
                      const preview = e.currentTarget.parentElement?.querySelector<HTMLElement>("[data-drag-preview]");
                      if (preview) e.dataTransfer.setDragImage(preview, 20, 18);
                    }}
                    onDragEnd={() => onDragKind(null)}
                    onClick={() => onAdd(b.kind)}
                    title={builtBy}
                    className="flex w-full items-center gap-3 rounded-lg border border-slate-800 bg-slate-900 px-3 py-2 text-left hover:border-teal-500/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-teal-300">
                    <span
                      className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-md border-[1.5px] ${
                        box ? "border-dashed border-teal-500/80" : "border-teal-500/40"
                      }`}>
                      <KindIcon kind={b.kind} className="h-4 w-4 text-teal-300" />
                    </span>
                    <span className="min-w-0">
                      <span className="block truncate text-sm font-medium text-white">{b.title}</span>
                      {b.summary && <span className="block truncate text-[11px] text-slate-400">{b.summary}</span>}
                      <span className="sr-only">{builtBy}</span>
                    </span>
                  </button>
                  {/* The drag image: rendered off screen, because a browser can only draw a drag image from a rendered element. */}
                  <div
                    data-drag-preview
                    aria-hidden
                    className="pointer-events-none fixed -left-[9999px] top-0 flex min-w-[150px] items-center gap-2 rounded-lg border border-slate-600 bg-slate-900 px-3 py-2 text-white shadow-md">
                    <KindIcon kind={b.kind} className="h-5 w-5 shrink-0 text-teal-300" />
                    <span className="text-sm font-semibold">{b.title}</span>
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </div>
      <div className="border-t border-slate-800 px-3 py-2">
        <HowLinking links={links} />
      </div>
    </aside>
  );
}

// HowLinking opens every line's card in one popover: what each link means and
// why it helps. It closes on Escape or a click outside.
function HowLinking({ links }: { links: EdgeCard[] }) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    const onClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    window.addEventListener("keydown", onKey);
    window.addEventListener("mousedown", onClick);
    return () => {
      window.removeEventListener("keydown", onKey);
      window.removeEventListener("mousedown", onClick);
    };
  }, [open]);

  if (links.length === 0) return null;
  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        aria-expanded={open}
        aria-haspopup="dialog"
        onClick={() => setOpen((o) => !o)}
        className="inline-flex items-center gap-1 rounded-md px-1.5 py-1 text-[11px] text-slate-400 hover:bg-slate-800 hover:text-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-teal-300">
        <Link2 className="h-3.5 w-3.5" aria-hidden />
        How linking works
      </button>
      {open && (
        <div
          role="dialog"
          aria-label="How linking works"
          data-testid="how-linking"
          className="absolute bottom-9 left-0 z-50 max-h-[70vh] w-80 overflow-y-auto rounded-xl border border-slate-700 bg-slate-900 p-3 text-xs shadow-2xl">
          <div className="mb-2 flex items-center justify-between">
            <p className="text-slate-300">Drag from a block&apos;s dot onto another block.</p>
            <button type="button" aria-label="Close" onClick={() => setOpen(false)} className="rounded p-0.5 text-slate-400 hover:text-white">
              <X className="h-3.5 w-3.5" />
            </button>
          </div>
          <ul className="space-y-2">
            {links.map((l) => (
              <li key={`${l.source}>${l.target}`} className="rounded-lg border border-slate-800 bg-slate-950/60 px-3 py-2">
                <LinkCardBody card={l} />
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
