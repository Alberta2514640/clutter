"use client";

import { useDroppable } from "@dnd-kit/core";
import React from "react";
import { renderNodeLabel } from "../shared/constants";
import type { Edge, NodeInstance } from "../types";
import { CanvasConnections } from "./CanvasConnections";
import { CanvasNode } from "./CanvasNode";

export function Canvas({
  canvasRef,
  nodes,
  edges,
  connectingFrom,
  onNodeClick,
  onCanvasClick,
}: {
  canvasRef: React.RefObject<HTMLDivElement | null>;
  nodes: NodeInstance[];
  edges: Edge[];
  connectingFrom: string | null;
  onNodeClick: (id: string) => void;
  onCanvasClick: () => void;
}) {
  const { setNodeRef, isOver } = useDroppable({ id: "canvas" });

  return (
    <div className="mt-4 p-4 pt-0">
      <div
        ref={(el) => {
          canvasRef.current = el;
          setNodeRef(el);
        }}
        onClick={onCanvasClick}
        className={[
          "relative w-full overflow-hidden rounded-2xl border",
          "border-white/10 bg-white/5 backdrop-blur",
          "min-h-[72vh]",
          isOver ? "ring-2 ring-sky-400/40" : "",
        ].join(" ")}
      >
        <div className="absolute inset-0 pointer-events-none opacity-60 bg-[radial-gradient(circle,_rgba(148,163,184,0.35)_1px,_transparent_0)] bg-[length:24px_24px]" />
        <div className="absolute inset-0 pointer-events-none bg-gradient-to-b from-white/5 via-transparent to-black/20" />

        <CanvasConnections nodes={nodes} edges={edges} />

        {nodes.map((n) => (
          <CanvasNode
            key={n.id}
            node={n}
            label={renderNodeLabel(n.type)}
            selected={connectingFrom === n.id}
            onClick={() => onNodeClick(n.id)}
          />
        ))}
      </div>
    </div>
  );
}