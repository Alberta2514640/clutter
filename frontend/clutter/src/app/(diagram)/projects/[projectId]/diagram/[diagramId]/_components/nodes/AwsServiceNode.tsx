"use client";

import type { NodeData } from "@/lib/types";
import { Handle, Position, type Node, type NodeProps } from "@xyflow/react";

export default function AwsServiceNode({ data, selected }: NodeProps<Node<NodeData>>) {
  return (
    <div
      className={[
        "rounded-xl border px-4 py-3 shadow-lg transition-all",
        "bg-[rgba(30,35,45,0.85)]",
        selected
          ? "border-[rgba(100,180,255,0.8)] shadow-[0_0_20px_rgba(100,180,255,0.3)]"
          : "border-white/20",
      ].join(" ")}
    >
      <div className="flex items-center gap-3">
        <div className="grid h-10 w-10 shrink-0 place-items-center rounded-lg border border-white/20 bg-white/10 text-sm font-bold">
          {data.img}
        </div>
        <div className="leading-tight">
          <div className="text-sm font-semibold text-white">{data.label}</div>
        </div>
      </div>

      <Handle
        type="target"
        position={Position.Left}
        style={{
          width: 12,
          height: 12,
          borderRadius: 999,
          border: "2px solid rgba(100,180,255,0.5)",
          background: "rgba(30,35,45,0.95)",
          left: -6,
        }}
      />
      <Handle
        type="source"
        position={Position.Right}
        style={{
          width: 12,
          height: 12,
          borderRadius: 999,
          border: "2px solid rgba(100,180,255,0.5)",
          background: "rgba(30,35,45,0.95)",
          right: -6,
        }}
      />
    </div>
  );
}
