"use client";

import React from "react";

export function PaletteGroup({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section className="space-y-2">
      <div className="text-[11px] font-semibold uppercase tracking-wider text-slate-400/80">
        {title}
      </div>

      <div className="space-y-2">{children}</div>
    </section>
  );
}
