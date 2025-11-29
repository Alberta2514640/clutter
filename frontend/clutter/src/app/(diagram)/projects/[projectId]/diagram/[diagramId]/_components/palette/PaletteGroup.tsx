"use client";

import React from "react";

export function PaletteGroup({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-[11px] font-semibold uppercase tracking-wide text-slate-300/80">{title}</span>
      <div className="flex flex-wrap gap-2">{children}</div>
    </div>
  );
}