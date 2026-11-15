"use client";

import CloudMark from "@/components/clouds/CloudMark";
import type { BlockNode as BlockNodeType } from "@/lib/features/diagram/blocksDocument";
import type { NodeLook } from "@/lib/features/deployments/views";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { createContext, useContext } from "react";
import ServiceIcon from "./ServiceIcon";

// How each block looks in the current "View as" choice comes from the canvas,
// which provides it here so the node data holds only what the document saves.
export const BlockLooks = createContext<(block: { id: string; kind: string }) => NodeLook>((block) => ({
  subtitle: block.kind,
  iconSrc: null,
  dimmed: false,
  status: null,
  statusDetail: null,
  badges: [],
}));

export default function BlockNode({ id, data, selected }: NodeProps<BlockNodeType>) {
  const look = useContext(BlockLooks)({ id, kind: data.kind });
  return (
    <div
      data-testid="block-node"
      data-kind={data.kind}
      data-dimmed={look.dimmed || undefined}
      title={look.statusDetail ?? undefined}
      className={`min-w-[150px] max-w-[260px] rounded-lg border bg-slate-900 px-3 py-2 text-white shadow-md ${
        selected ? "border-teal-400 ring-2 ring-teal-400/40" : look.dimmed ? "border-dashed border-slate-600" : "border-slate-600"
      }`}>
      <Handle type="target" position={Position.Left} className="!h-3 !w-3 !border-slate-900 !bg-teal-400" />
      <div className={`flex items-center gap-2 ${look.dimmed ? "opacity-45" : ""}`}>
        <ServiceIcon src={look.iconSrc} kind={data.kind} className="h-5 w-5 shrink-0 text-teal-300" />
        <div className="min-w-0">
          <div className="truncate text-sm font-semibold">{data.name || "(no name)"}</div>
          <div className="truncate text-xs text-slate-400">{look.subtitle}</div>
        </div>
      </div>
      {look.status && (
        <div data-testid="block-status" className="mt-1.5 text-[11px] font-medium text-slate-300">
          {look.status}
          {look.statusDetail && <span className="sr-only">: {look.statusDetail}</span>}
        </div>
      )}
      {look.badges.length > 0 && (
        <div className="mt-1.5 flex items-center gap-1.5">
          {look.badges.map((b) => (
            <CloudMark key={b.cloud.connector} cloud={b.cloud} size={16} off={!b.on} label={b.detail} />
          ))}
        </div>
      )}
      <Handle type="source" position={Position.Right} className="!h-3 !w-3 !border-slate-900 !bg-teal-400" />
    </div>
  );
}
