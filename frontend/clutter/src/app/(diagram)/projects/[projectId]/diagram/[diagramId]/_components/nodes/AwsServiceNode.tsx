// src/app/.../AwsServiceNode.tsx
"use client";

import { useDiagramActions } from "@/lib/stores/diagramStore";
import type { NodeData } from "@/lib/types";
import {
  Handle,
  NodeToolbar,
  Position,
  type Node,
  type NodeProps,
} from "@xyflow/react";

export default function AwsServiceNode({
  id,
  data,
  selected,
}: NodeProps<Node<NodeData>>) {
  const { updateNodeData } = useDiagramActions();

  const handleLabelChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    updateNodeData(id, { label: e.target.value });
  };

  return (
    <>
      {/* Floating per-node settings toolbar */}
      <NodeToolbar
        // default behavior: visible when node is selected
        position={Position.Top}
        offset={12}
        className="rounded-lg border border-slate-800 bg-slate-950/95 px-3 py-2 shadow-xl"
      >
        <div className="space-y-2 text-[11px] text-slate-200">
          <div className="flex items-center gap-2">
            <div className="grid h-6 w-6 place-items-center rounded-md border border-slate-700 bg-slate-900 text-[10px] font-bold">
              {data.img}
            </div>
            <span className="text-[10px] uppercase tracking-wide text-slate-400">
              {data.label || "Block settings"}
            </span>
          </div>

          <label className="flex flex-col gap-1">
            <span>Display label</span>
            <input
              className="rounded-md border border-slate-700 bg-slate-900 px-2 py-1 text-[11px] text-white outline-none focus:border-teal-500 focus:ring-1 focus:ring-teal-500"
              value={data.label ?? ""}
              onChange={handleLabelChange}
              placeholder="Lambda name, table name..."
            />
          </label>

          {/* Future: env vars / advanced config */}
          {/* e.g. add a button here later to open a modal/advanced editor */}
        </div>
      </NodeToolbar>

      {/* Node body */}
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
            <div className="text-sm font-semibold text-white">
              {data.label}
            </div>
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
    </>
  );
}