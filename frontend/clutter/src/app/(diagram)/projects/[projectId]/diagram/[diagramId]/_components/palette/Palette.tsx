"use client";

import React, { useMemo } from "react";
import { GROUPS, PALETTE_ITEMS } from "../shared/constants";
import { NodeIcon } from "../shared/NodeIcon";
import type { PaletteGroup as PaletteGroupType, PaletteItem } from "../types";
import { PaletteChip } from "./PaletteChip";
import { PaletteGroup } from "./PaletteGroup";

function groupPalette(items: PaletteItem[]) {
  const by: Record<PaletteGroupType, PaletteItem[]> = { General: [], Storage: [], Compute: [], Network: [] };
  for (const it of items) by[it.group].push(it);
  return by;
}

export function Palette({ mounted }: { mounted: boolean }) {
  const grouped = useMemo(() => groupPalette(PALETTE_ITEMS), []);

  if (!mounted) {
    return (
      <div className="flex flex-wrap gap-3 opacity-80">
        {GROUPS.map((g) => (
          <PaletteGroup key={g} title={g}>
            {grouped[g].map((it) => (
              <div
                key={it.type}
                className="flex items-center gap-2 rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-xs text-slate-100"
              >
                <NodeIcon type={it.type} />
                <span>{it.label}</span>
              </div>
            ))}
          </PaletteGroup>
        ))}
      </div>
    );
  }

  return (
    <div className="flex flex-wrap gap-3">
      {GROUPS.map((g) => (
        <PaletteGroup key={g} title={g}>
          {grouped[g].map((it) => (
            <PaletteChip key={it.type} item={it} />
          ))}
        </PaletteGroup>
      ))}
    </div>
  );
}