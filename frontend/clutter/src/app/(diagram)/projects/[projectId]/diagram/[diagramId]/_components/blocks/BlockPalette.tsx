"use client";

import { cloudLook } from "@/lib/features/catalog/display";
import type { PaletteBlock } from "@/lib/features/catalog/forms";
import type { CatalogEntry } from "@/lib/features/catalog/types";
import { Loader2, X } from "lucide-react";
import KindIcon from "./KindIcon";

export const BLOCK_DND_MIME = "application/x-clutter-block";

type Props = {
  blocks: PaletteBlock[];
  catalog: CatalogEntry[];
  loading: boolean;
  error: string | null;
  onAdd: (kind: string) => void;
  blockTitle: (kind: string) => string;
};

// The palette lists the blocks the connected clouds can build, read from the
// manifests. Drag a block onto the canvas, or click it to add it.
export default function BlockPalette({ blocks, catalog, loading, error, onAdd, blockTitle }: Props) {
  // Every cloud is listed under every block, so a cloud that cannot build one shows as such.
  const allClouds = catalog.map((c) => cloudLook(c.manifest));
  const edges = new Map<string, string>();
  for (const { manifest } of catalog) {
    for (const e of manifest.edges ?? []) {
      const key = `${e.source}>${e.target}`;
      if (!edges.has(key)) edges.set(key, `${blockTitle(e.source)} to ${blockTitle(e.target)}: ${e.meaning}`);
    }
  }

  return (
    <aside aria-label="Building blocks" className="flex h-full w-60 shrink-0 flex-col border-r border-slate-800 bg-slate-950">
      <div className="border-b border-slate-800 px-4 py-3">
        <h2 className="text-sm font-semibold text-white">Building blocks</h2>
        <p className="mt-0.5 text-xs text-slate-400">Drag onto the canvas, or click to add.</p>
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
              return (
                <li key={b.kind}>
                  <button
                    type="button"
                    draggable
                    data-testid={`palette-${b.kind}`}
                    onDragStart={(e) => {
                      e.dataTransfer.setData(BLOCK_DND_MIME, b.kind);
                      e.dataTransfer.effectAllowed = "move";
                    }}
                    onClick={() => onAdd(b.kind)}
                    title={`Built by ${b.clouds.map((c) => c.title).join(", ")}`}
                    className="flex w-full items-center gap-3 rounded-lg border border-slate-800 bg-slate-900 px-3 py-2 text-left hover:border-teal-500/60">
                    <KindIcon kind={b.kind} className="h-5 w-5 shrink-0 text-teal-300" />
                    <span className="min-w-0">
                      <span className="block text-sm font-medium text-white">{b.title}</span>
                      <span className="mt-1 flex flex-wrap items-center gap-1" aria-hidden>
                        {allClouds.map((c) => {
                          const on = b.clouds.some((x) => x.connector === c.connector);
                          return (
                            <span
                              key={c.connector}
                              title={on ? `Built by ${c.title}` : `Not available on ${c.title}`}
                              className={`inline-flex items-center gap-0.5 rounded border px-1 py-px text-[10px] leading-tight ${
                                on ? "border-slate-700 text-slate-300" : "border-dashed border-slate-800 text-slate-500"
                              }`}>
                              {!on && <X className="h-2.5 w-2.5" aria-hidden />}
                              <span className={on ? undefined : "line-through"}>{c.shortName}</span>
                            </span>
                          );
                        })}
                      </span>
                      <span className="sr-only">Built by {b.clouds.map((c) => c.title).join(", ")}</span>
                    </span>
                  </button>
                </li>
              );
            })}
          </ul>
        )}
        {edges.size > 0 && (
          <div className="mt-5">
            <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">Links you can draw</h3>
            <ul className="mt-2 space-y-1 text-xs text-slate-400">
              {[...edges.values()].map((t) => (
                <li key={t}>{t}</li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </aside>
  );
}
