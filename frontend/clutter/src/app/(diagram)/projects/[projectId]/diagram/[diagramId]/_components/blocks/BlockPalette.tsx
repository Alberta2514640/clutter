"use client";

import CloudLogo from "@/components/common/CloudLogo";
import ErrorState from "@/components/common/ErrorState";
import { Skeleton } from "@/components/ui/skeleton";
import { kindBadgeClass, kindIconClass, type CloudLook } from "@/lib/features/catalog/display";
import type { PaletteBlock } from "@/lib/features/catalog/forms";
import { NETWORK_KIND } from "@/lib/features/diagram/blocksDocument";
import { cn } from "@/lib/utils";
import { Link2, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import type { EdgeCard } from "./canvasContext";
import { LinkCardBody } from "./FloatingEdge";
import KindIcon from "./KindIcon";

export const BLOCK_DND_MIME = "application/x-clutter-block";

type Props = {
  blocks: PaletteBlock[];
  loading: boolean;
  error: string | null;
  onRetry: () => void;
  // The clouds the organization has a connection to: a block one of them
  // cannot build says so ("Not on AWS").
  linked: CloudLook[];
  onAdd: (kind: string) => void;
  // Called with the kind a drag starts with, and null when it ends.
  onDragKind: (kind: string | null) => void;
  // Every line the clouds in play draw, for "How linking works".
  links: EdgeCard[];
};

// The palette lists the blocks the clouds can build, read from the
// manifests: each with its icon, its name and one short line on what it is
// for. Click it (or press Enter) to add it near the middle of the view, or
// drag it onto the canvas. A block a linked cloud cannot build carries
// "Not on (cloud)".
export default function BlockPalette({ blocks, loading, error, onRetry, linked, onAdd, onDragKind, links }: Props) {
  return (
    <aside aria-label="Building blocks" className="bg-background flex h-full w-64 shrink-0 flex-col border-r">
      <div className="space-y-0.5 border-b px-4 pt-3 pb-2">
        <h2 className="text-sm font-semibold">Building blocks</h2>
        <p className="text-muted-foreground text-xs">Click to add, or drag onto the canvas.</p>
        <div className="-ml-1.5 pt-1">
          <HowLinking links={links} />
        </div>
      </div>
      <div className="flex-1 overflow-y-auto p-3">
        {loading ? (
          <div role="status" className="space-y-2">
            <span className="sr-only">Loading blocks…</span>
            {Array.from({ length: 6 }, (_, i) => (
              <Skeleton key={i} className="h-14" />
            ))}
          </div>
        ) : error ? (
          <ErrorState variant="inline" description={`Could not load the blocks. ${error}`} onRetry={onRetry} />
        ) : blocks.length === 0 ? (
          <p className="text-muted-foreground text-sm">No cloud offers any blocks yet.</p>
        ) : (
          <ul className="space-y-1.5">
            {blocks.map((b) => {
              const missing = linked.filter((l) => !b.clouds.some((c) => c.connector === l.connector));
              const builtBy = b.clouds.filter((c) => linked.some((l) => l.connector === c.connector));
              const notOn = missing.length > 0 ? ` Not on ${missing.map((m) => m.title).join(" or ")}.` : "";
              const tip =
                linked.length === 0
                  ? `${b.title}. Built by ${b.clouds.map((c) => c.title).join(", ")}.`
                  : builtBy.length > 0
                    ? `Built by ${builtBy.map((c) => c.title).join(", ")}.${notOn}`
                    : `None of your connected clouds builds it.${notOn}`;
              // No linked cloud builds it: a quiet, dimmed row, still addable.
              const unavailable = linked.length > 0 && builtBy.length === 0;
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
                    title={tip}
                    className="group surface-card surface-card-interactive focus-visible:ring-ring/50 flex w-full cursor-grab items-center gap-3 rounded-lg px-3 py-2 text-left outline-none focus-visible:ring-[3px] active:cursor-grabbing">
                    <span
                      className={cn(
                        kindBadgeClass(b.kind),
                        "flex size-8 shrink-0 items-center justify-center rounded-md",
                        box && "border-tint-violet-fg/60 border-[1.5px] border-dashed bg-transparent",
                        unavailable && "opacity-55",
                      )}>
                      <KindIcon kind={b.kind} className="size-4" />
                    </span>
                    <span className={cn("min-w-0 flex-1", unavailable && "opacity-55")}>
                      <span className="block truncate text-sm font-medium">{b.title}</span>
                      {b.summary && <span className="text-muted-foreground block truncate text-[11px]">{b.summary}</span>}
                      <span className="sr-only">{tip}</span>
                    </span>
                    {missing.length > 0 && (
                      <span data-testid="not-on-cloud" className="flex shrink-0 items-center gap-1">
                        {missing.map((m) => (
                          <CloudLogo key={m.connector} connector={m.connector} shortName={m.shortName} src={m.logoSrc} size={14} off />
                        ))}
                      </span>
                    )}
                  </button>
                  {/* The drag image: rendered off screen, because a browser can only draw a drag image from a rendered element. */}
                  <div
                    data-drag-preview
                    aria-hidden
                    className="surface-raised text-popover-foreground pointer-events-none fixed top-0 -left-[9999px] flex min-w-[160px] items-center gap-2 rounded-lg border px-3 py-2">
                    <KindIcon kind={b.kind} className={cn(kindIconClass(b.kind), "size-5 shrink-0")} />
                    <span className="text-sm font-semibold">{b.title}</span>
                  </div>
                </li>
              );
            })}
          </ul>
        )}
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
        className="text-muted-foreground hover:bg-accent hover:text-foreground focus-visible:ring-ring/50 inline-flex h-7 items-center gap-1 rounded-md px-1.5 text-xs outline-none focus-visible:ring-[3px]">
        <Link2 className="size-3.5" aria-hidden />
        How linking works
      </button>
      {open && (
        <div
          role="dialog"
          aria-label="How linking works"
          data-testid="how-linking"
          className="surface-raised text-popover-foreground absolute top-8 left-0 z-50 max-h-[70vh] w-80 overflow-y-auto rounded-xl border p-3 text-xs animate-fade-in">
          <div className="mb-2 flex items-center justify-between">
            <p className="text-muted-foreground">Drag from a block&apos;s dot onto another block.</p>
            <button
              type="button"
              aria-label="Close"
              onClick={() => setOpen(false)}
              className="text-muted-foreground hover:text-foreground hover:bg-accent inline-flex size-6 items-center justify-center rounded-md">
              <X className="size-3.5" />
            </button>
          </div>
          <ul className="space-y-2">
            {links.map((l) => (
              <li key={`${l.source}>${l.target}`} className="bg-background rounded-lg border px-3 py-2">
                <LinkCardBody card={l} />
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
