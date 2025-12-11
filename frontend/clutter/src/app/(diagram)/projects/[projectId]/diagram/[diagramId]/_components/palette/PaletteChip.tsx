"use client";

import { useDraggable } from "@dnd-kit/core";
import { CSS } from "@dnd-kit/utilities";
import React from "react";
import { NodeIcon } from "../shared/NodeIcon";
import type { PaletteItem } from "../types";

export function PaletteChip({ item }: { item: PaletteItem }) {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({
    id: `palette:${item.type}`,
    data: { kind: "palette", nodeType: item.type },
  });

  const style: React.CSSProperties = {
    transform: CSS.Translate.toString(transform),
    opacity: isDragging ? 0 : 1,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      {...listeners}
      {...attributes}
      className="select-none cursor-grab active:cursor-grabbing flex items-center gap-2 rounded-xl border px-3 py-2 text-xs border-white/10 bg-white/5 text-slate-100 hover:bg-white/10"
    >
      <NodeIcon type={item.type} />
      <span>{item.label}</span>
    </div>
  );
}