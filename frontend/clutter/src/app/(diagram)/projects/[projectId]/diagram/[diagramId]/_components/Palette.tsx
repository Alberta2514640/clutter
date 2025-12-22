"use client";

import type { PaletteItem } from "@/lib/types";
import { ChevronLeft, ChevronRight, Layers } from "lucide-react";
import React, { useMemo, useState } from "react";

const DND_MIME = "application/x-palette-item";

type Section = { title: string; items: PaletteItem[] };

export default function Palette() {
  const [isCollapsed, setIsCollapsed] = useState(false);

  const sections = useMemo<Section[]>(() => {
    const storage: PaletteItem[] = [
      { label: "DynamoDB", img: "DB" },
      { label: "S3", img: "S3" },
    ];
    const compute: PaletteItem[] = [
      { label: "Lambda", img: "λ" },
      { label: "EC2 Container", img: "EC2" },
    ];
    const network: PaletteItem[] = [{ label: "API Gateway", img: "API" }];

    return [
      { title: "COMPUTE", items: compute },
      { title: "STORAGE", items: storage },
      { title: "NETWORK", items: network },
    ];
  }, []);

  return (
    <aside
      className={[
        "relative h-full shrink-0 border-r border-slate-800 bg-slate-950/70 backdrop-blur",
        "transition-[width] duration-200",
        isCollapsed ? "w-14" : "w-[300px]",
      ].join(" ")}
    >
      {/* Mid-edge circular collapse/expand button (like before) */}
      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation();
          setIsCollapsed((v) => !v);
        }}
        className={[
          "absolute -right-3 top-1/2 -translate-y-1/2 z-20",
          "grid h-7 w-7 place-items-center rounded-full",
          "border border-slate-700 bg-slate-900 shadow-lg",
          "hover:bg-slate-800 transition",
        ].join(" ")}
        aria-label={isCollapsed ? "Expand palette" : "Collapse palette"}
        title={isCollapsed ? "Expand" : "Collapse"}
      >
        {isCollapsed ? (
          <ChevronRight className="h-4 w-4 text-slate-300" />
        ) : (
          <ChevronLeft className="h-4 w-4 text-slate-300" />
        )}
      </button>

      {/* Header */}
      <div className="flex h-14 items-center gap-2 border-b border-slate-800 px-3">
        <div className="flex items-center gap-2 min-w-0">
          <div className="grid h-8 w-8 place-items-center rounded-lg border border-slate-800 bg-slate-900">
            <Layers className="h-4 w-4 text-slate-300" />
          </div>

          {!isCollapsed && (
            <div className="min-w-0">
              <div className="truncate text-sm font-semibold text-slate-100">
                Nodes & Edges
              </div>
              <div className="truncate text-[11px] text-slate-400">
                Drag items onto the canvas
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Collapsed rail label */}
      {isCollapsed ? (
        <div className="flex h-[calc(100%-3.5rem)] items-center justify-center">
          <span className="select-none text-[11px] font-semibold tracking-widest text-slate-300 [writing-mode:vertical-rl] rotate-180">
            PALETTE
          </span>
        </div>
      ) : (
        <>
          {/* Body */}
          <div className="h-[calc(100%-3.5rem)] overflow-y-auto px-3 py-3">
            <div className="mb-3 rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-2 text-[11px] text-slate-300">
              Tip: connect nodes by dragging from the handles.
            </div>

            <div className="space-y-5">
              {sections.map((sec) => (
                <section key={sec.title}>
                  <div className="mb-2 flex items-center gap-2">
                    <div className="text-[11px] font-semibold tracking-wider text-slate-400">
                      {sec.title}
                    </div>
                    <div className="h-px flex-1 bg-slate-800" />
                  </div>

                  <div className="space-y-2">
                    {sec.items.map((item) => (
                      <div
                        key={`${sec.title}-${item.label}`}
                        draggable
                        onDragStart={(e) => {
                          e.dataTransfer.setData(DND_MIME, JSON.stringify(item));
                          e.dataTransfer.effectAllowed = "move";
                        }}
                        className={[
                          "flex cursor-grab select-none items-center gap-3 rounded-lg",
                          "border border-slate-800 bg-slate-900/40 px-3 py-2.5",
                          "hover:bg-slate-900/70 hover:border-slate-700 transition",
                          "active:cursor-grabbing",
                        ].join(" ")}
                      >
                        <div className="grid h-8 w-8 shrink-0 place-items-center rounded-md border border-slate-800 bg-slate-950 text-xs font-bold text-slate-200">
                          {item.img}
                        </div>
                        <div className="min-w-0">
                          <div className="truncate text-sm font-medium text-slate-100">
                            {item.label}
                          </div>
                          <div className="truncate text-[11px] text-slate-400">
                            Drag to add
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </section>
              ))}
            </div>
          </div>

          {/* Footer */}
          <div className="h-10 border-t border-slate-800 px-3 flex items-center justify-between text-[11px] text-slate-400">
            <span>Canvas</span>
            <span className="text-slate-500">v1</span>
          </div>
        </>
      )}
    </aside>
  );
}