"use client";

import React, { useMemo } from "react";
import { GROUPS, PALETTE_ITEMS } from "../shared/constants";
import { NodeIcon } from "../shared/NodeIcon";
import type { PaletteGroup as PaletteGroupType, PaletteItem } from "../types";
import { PaletteChip } from "./PaletteChip";
import { PaletteGroup } from "./PaletteGroup";

function groupPalette(items: PaletteItem[]) {
  const by: Record<PaletteGroupType, PaletteItem[]> = {
    General: [],
    Storage: [],
    Compute: [],
    Network: [],
  };
  for (const it of items) by[it.group].push(it);
  return by;
}

export function Palette({ mounted }: { mounted: boolean }) {
  const grouped = useMemo(() => groupPalette(PALETTE_ITEMS), []);

  return (
    <aside
      className={[
        "shrink-0",
        "w-fit min-w-[200px] max-w-[240px]",
        "sticky top-24 self-start",
        "rounded-2xl border border-white/10 bg-white/5 backdrop-blur",
        "shadow-[0_10px_30px_rgba(0,0,0,0.35)]",
        // match canvas size
        "h-[72vh] min-h-[72vh]",
        // layout
        "flex flex-col",
      ].join(" ")}
    >
      {/* stacked subtitle helper text */}
      <div className="px-4 pt-4 pb-3 border-b border-white/10">
        <div className="text-[11px] text-slate-300/80 leading-tight">
          Drag components onto the canvas
        </div>
        <div className="mt-1 text-[11px] text-slate-400/80 leading-tight">
          Drag → Drop
        </div>
      </div>

      {/* Scroll body fills remaining height */}
      <div className="relative flex-1 min-h-0">
        <div className="h-full overflow-auto px-4 py-4 space-y-5">
          {GROUPS.map((g) => (
            <PaletteGroup key={g} title={g}>
              <div className="flex flex-col gap-2">
                {!mounted
                  ? grouped[g].map((it) => (
                      <div
                        key={it.type}
                        className={[
                          "flex items-center gap-2",
                          "rounded-xl border border-white/10",
                          "bg-white/[0.06] hover:bg-white/[0.09]",
                          "px-3 py-2 text-xs text-slate-100",
                          "transition-colors",
                        ].join(" ")}
                      >
                        <NodeIcon type={it.type} />
                        <span className="truncate">{it.label}</span>
                      </div>
                    ))
                  : grouped[g].map((it) => <PaletteChip key={it.type} item={it} />)}
              </div>
            </PaletteGroup>
          ))}
        </div>

        <div className="pointer-events-none absolute inset-x-0 bottom-0 h-10 bg-gradient-to-t from-black/25 to-transparent rounded-b-2xl" />
      </div>
    </aside>
  );
}