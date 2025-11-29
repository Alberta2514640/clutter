"use client";

import React from "react";
import { NODE_HEIGHT, NODE_WIDTH } from "../shared/constants";
import type { Edge, NodeInstance } from "../types";

function center(n: NodeInstance) {
  return { x: n.x + NODE_WIDTH / 2, y: n.y + NODE_HEIGHT / 2 };
}

export function CanvasConnections({ nodes, edges }: { nodes: NodeInstance[]; edges: Edge[] }) {
  return (
    <svg className="absolute inset-0 pointer-events-none">
      <defs>
        <marker id="arrowhead" markerWidth="8" markerHeight="8" refX="4" refY="4" orient="auto">
          <path d="M0,0 L8,4 L0,8 z" fill="rgba(148,163,184,0.9)" />
        </marker>
      </defs>

      {edges.map((e) => {
        const from = nodes.find((n) => n.id === e.fromId);
        const to = nodes.find((n) => n.id === e.toId);
        if (!from || !to) return null;

        const a = center(from);
        const b = center(to);

        return (
          <line
            key={e.id}
            x1={a.x}
            y1={a.y}
            x2={b.x}
            y2={b.y}
            stroke="rgba(148,163,184,0.85)"
            strokeWidth={2}
            markerEnd="url(#arrowhead)"
          />
        );
      })}
    </svg>
  );
}