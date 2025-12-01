"use client";

import { useDraggable } from "@dnd-kit/core";
import { CSS } from "@dnd-kit/utilities";
import React from "react";
import { NODE_HEIGHT, NODE_WIDTH } from "../shared/constants";
import { NodeIcon } from "../shared/NodeIcon";
import type { NodeInstance } from "../types";

export function CanvasNode({
  node,
  label,
  selected,
  onClick,
}: {
  node: NodeInstance;
  label: string;
  selected: boolean;
  onClick: () => void;
}) {
  const { attributes, listeners, setNodeRef, transform } = useDraggable({
    id: node.id,
    data: { kind: "node" },
  });

  // Real node moves while dragging (no overlay for nodes)
  const style: React.CSSProperties = {
    width: NODE_WIDTH,
    height: NODE_HEIGHT,
    transform: `translate(${node.x}px, ${node.y}px) ${transform ? CSS.Translate.toString(transform) : ""}`,
  };

  return (
    <div ref={setNodeRef} style={style} className="absolute z-10 select-none">
      <div
        data-node-id={node.id}
        {...listeners}
        {...attributes}
        onClick={(e) => {
          e.stopPropagation();
          onClick();
        }}
        className={[
          "w-full h-full px-3 py-2 rounded-xl border flex items-center gap-2 text-xs cursor-move",
          "bg-slate-950/60 backdrop-blur shadow-xl",
          selected ? "border-sky-400/60 ring-2 ring-sky-400/30" : "border-white/10 hover:border-white/20",
        ].join(" ")}
      >
        <NodeIcon type={node.type} />
        <span className="font-medium text-slate-100">{label}</span>
      </div>
    </div>
  );
}