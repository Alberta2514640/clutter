"use client";
import type { PaletteItem } from "@/lib/types";
import { ChevronLeft, ChevronRight } from "lucide-react";
import React, { useMemo, useState } from "react";

const DND_MIME = "application/x-palette-item";

export default function Palette() {
  const [isCollapsed, setIsCollapsed] = useState(false);

  const sections = useMemo(() => {
    const storage: PaletteItem[] = [
      { label: "DynamoDB", img: "DB" },
      { label: "S3", img: "S3" },
    ];
    const compute: PaletteItem[] = [
      { label: "Lambda", img: "λ" },
      { label: "EC2 Container", img: "EC2" }
    ];
    const network: PaletteItem[] = [{ label: "API Gateway", img: "API" }];
    return [
      { title: "COMPUTE", items: compute },
      { title: "STORAGE", items: storage },
      { title: "NETWORK", items: network },
    ] as const;
  }, []);

  const handleExpand = () => {
    if (isCollapsed) {
      setIsCollapsed(false);
    }
  };

  const handleToggle = (e: React.MouseEvent) => {
    e.stopPropagation();
    setIsCollapsed(!isCollapsed);
  };

  return (
    <aside 
      onClick={handleExpand}
      className={`relative rounded-xl border border-slate-800 bg-slate-900/60 shadow-lg backdrop-blur-sm transition-all duration-300 ${
        isCollapsed ? 'w-16 cursor-pointer' : 'w-[270px]'
      }`}
    >
      {/* Collapse/Expand Button - Centered on the right edge */}
      <button
        onClick={handleToggle}
        className="absolute -right-3 top-1/2 -translate-y-1/2 z-10 p-1.5 rounded-full bg-slate-800 border border-slate-700 hover:bg-slate-700 transition-colors shadow-lg"
        title={isCollapsed ? "Expand palette" : "Collapse palette"}
      >
        {isCollapsed ? (
          <ChevronRight className="w-4 h-4 text-gray-400" />
        ) : (
          <ChevronLeft className="w-4 h-4 text-gray-400" />
        )}
      </button>

      {/* Vertical "PALETTE" text when collapsed */}
      {isCollapsed && (
        <div className="flex items-center justify-center py-8">
          <span className="text-base font-semibold tracking-wider [writing-mode:vertical-lr] rotate-180">
            PALETTE
          </span>
        </div>
      )}

      {!isCollapsed && (
        <div className="p-4">
          <div className="mb-4 text-xs text-gray-400">
            Drag components onto the canvas
            <br />
            Connect nodes by dragging from handles
          </div>
          <div className="space-y-4">
            {sections.map((sec) => (
              <div key={sec.title} onClick={(e) => e.stopPropagation()}>
                <div className="mb-2 text-[11px] font-semibold tracking-wider text-gray-500">
                  {sec.title}
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
                      className="flex cursor-grab select-none items-center gap-3 rounded-lg border border-slate-700 bg-slate-800/50 px-3 py-2.5 transition-all hover:border-teal-500/50 hover:bg-slate-800 active:cursor-grabbing"
                    >
                      <div className="grid h-8 w-8 shrink-0 place-items-center rounded-md border border-slate-700 bg-slate-900 text-xs font-bold text-gray-300">
                        {item.img}
                      </div>
                      <div className="text-sm font-medium text-white">{item.label}</div>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </aside>
  );
}