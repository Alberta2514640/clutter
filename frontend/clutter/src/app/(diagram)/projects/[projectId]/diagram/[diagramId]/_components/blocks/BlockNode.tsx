"use client";

import CloudMark from "@/components/clouds/CloudMark";
import { VM_KIND, type BlockNode as BlockNodeType } from "@/lib/features/diagram/blocksDocument";
import { Handle, Position, useConnection, type NodeProps } from "@xyflow/react";
import { useId } from "react";
import { useCanvas } from "./canvasContext";
import ServiceIcon from "./ServiceIcon";

// One block on B's canvas. A line starts at the dot on its right. While a
// line is drawn the whole card is its target: a block the line may reach
// glows, the others dim (canvasContext.linkState).
export default function BlockNode({ id, data, selected }: NodeProps<BlockNodeType>) {
  const canvas = useCanvas();
  const look = canvas.look({ id, kind: data.kind });
  const connecting = useConnection((c) => c.inProgress);
  const link = connecting ? canvas.linkState(id) : null;
  const problems = canvas.problems(id);
  const tipId = useId();

  const ring = selected
    ? "border-teal-400 ring-2 ring-teal-400/40"
    : link === "valid"
      ? "border-teal-300 ring-4 ring-teal-400/30"
      : look.dimmed
        ? "border-dashed border-slate-600"
        : "border-slate-600";

  return (
    <div
      data-testid="block-node"
      data-kind={data.kind}
      data-dimmed={look.dimmed || undefined}
      data-link={link ?? undefined}
      title={look.statusDetail ?? undefined}
      className={`relative min-w-[150px] max-w-[260px] rounded-lg border bg-slate-900 px-3 py-2 text-white shadow-md transition-opacity ${ring} ${
        link === "invalid" ? "opacity-35" : ""
      }`}>
      {/* The target: the whole card, and only while a line is drawn, so it never gets in the way of a drag. */}
      <Handle
        type="target"
        position={Position.Left}
        isConnectableStart={false}
        className={`!absolute !inset-0 !h-full !w-full !transform-none !rounded-lg !border-0 !bg-transparent !opacity-0 ${
          connecting ? "" : "!pointer-events-none"
        }`}
      />
      <div className={`flex items-center gap-2 ${look.dimmed ? "opacity-45" : ""}`}>
        <ServiceIcon src={look.iconSrc} kind={data.kind} className="h-5 w-5 shrink-0 text-teal-300" />
        <div className="min-w-0">
          <div className="truncate text-sm font-semibold">{data.name || "(no name)"}</div>
          <div className="truncate text-xs text-slate-400">{look.subtitle}</div>
        </div>
      </div>
      {data.kind === VM_KIND && (
        <span
          className={`mt-1.5 inline-block rounded-full border px-1.5 text-[10px] leading-4 ${
            data.fields.public === true ? "border-teal-700 text-teal-300" : "border-slate-700 text-slate-400"
          }`}>
          {data.fields.public === true ? "public" : "private"}
        </span>
      )}
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
      {problems.length > 0 && (
        <span className="group/problem absolute -right-2 -top-2 z-10">
          <button
            type="button"
            data-testid="problem-badge"
            aria-label={`${data.name || "This block"} has a problem`}
            aria-describedby={tipId}
            className="nodrag flex h-4 w-4 items-center justify-center rounded-full bg-amber-400 text-[11px] font-bold leading-none text-slate-950 shadow focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-200">
            !
          </button>
          <span
            role="tooltip"
            id={tipId}
            className="pointer-events-none invisible absolute left-5 top-0 w-64 space-y-1 rounded-lg border border-amber-700/60 bg-slate-900 px-3 py-2 text-xs leading-snug text-slate-200 opacity-0 shadow-xl transition-opacity group-focus-within/problem:visible group-focus-within/problem:opacity-100 group-hover/problem:visible group-hover/problem:opacity-100">
            {problems.map((p) => (
              <span key={p} className="block">
                {p}
              </span>
            ))}
          </span>
        </span>
      )}
      <Handle type="source" position={Position.Right} className="!h-3 !w-3 !border-slate-900 !bg-teal-400" />
    </div>
  );
}
