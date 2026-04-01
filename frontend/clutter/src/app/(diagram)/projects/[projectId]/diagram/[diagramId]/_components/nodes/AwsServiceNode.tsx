"use client";

import { NodeData } from "@/lib/features/diagram/types";
import { useSupportedResources } from "@/lib/features/resources/hooks";
import { Handle, Position, type Node, type NodeProps } from "@xyflow/react";
import { AlertTriangle } from "lucide-react";
import Image from "next/image";
import { useMemo } from "react";

const RESOURCE_NAME_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;
const HIDDEN_VARIABLE_NAMES = new Set([
  "label", "position", "x", "y",
  "position_x", "position_y", "x_position", "y_position",
  "pos_x", "pos_y",
]);

function getVariableError(name: string, value: unknown, required: boolean): string | null {
  const stringValue = typeof value === "string" ? value.trim() : "";

  if (required) {
    const isMissing =
      value === undefined ||
      value === null ||
      (typeof value === "string" && stringValue === "");
    if (isMissing) return "This field is required.";
  }

  if (name === "resource_name" && stringValue && !RESOURCE_NAME_PATTERN.test(stringValue)) {
    return "Invalid format.";
  }

  return null;
}

export default function AwsServiceNode({ data, selected }: NodeProps<Node<NodeData>>) {
  const { data: supportedResources } = useSupportedResources();

  const validationErrors = useMemo(() => {
    const resourceDef = supportedResources?.find((r) => r.label === data.label);
    if (!resourceDef) return [];

    return resourceDef.variables
      .filter((v) => !HIDDEN_VARIABLE_NAMES.has(v.name.toLowerCase()))
      .flatMap((v) => {
        const error = getVariableError(v.name, data.variables?.[v.name], v.required);
        return error ? [{ name: v.name, error }] : [];
      });
  }, [data.label, data.variables, supportedResources]);

  const hasErrors = validationErrors.length > 0;

  const hasAnsiblePlaybook = Boolean(data.ansiblePlaybookName);
  const isEc2Node = data.img.includes("ec2");
  const hasQueuedAnsibleJob = isEc2Node && Boolean(data.lastAnsibleJobId);

  return (
    <div
      className={[
        "rounded-xl border px-4 py-3 shadow-lg transition-all",
        "bg-[rgba(30,35,45,0.85)]",
        selected
          ? "border-[rgba(100,180,255,0.8)] shadow-[0_0_20px_rgba(100,180,255,0.3)]"
          : hasErrors
            ? "border-amber-500/60 shadow-[0_0_12px_rgba(245,158,11,0.15)]"
            : "border-white/20",
      ].join(" ")}
    >
      <div className="flex items-center gap-3">
        <div className="grid h-10 w-10 shrink-0 place-items-center rounded-lg border border-white/20 bg-white/10 text-sm font-bold">
          <Image src={data.img as string} alt="" width={24} height={24} unoptimized />
        </div>

        <div className="min-w-0 flex-1 leading-tight">
          <div className="flex items-center gap-1.5">
            <div className="truncate text-sm font-semibold text-white">{data.label}</div>
            {hasErrors && !selected && (
              <AlertTriangle className="h-3.5 w-3.5 shrink-0 text-amber-400" />
            )}
          </div>

          {data.variables?.resource_name !== undefined && (
            <div className="mt-0.5 text-[11px] text-slate-400">
              {(() => {
                const name = String(data.variables.resource_name);
                return name.length > 32 ? `${name.slice(0, 32)}…` : name;
              })()}
            </div>
          )}

          {/* Required field errors — shown when node is not selected */}
          {hasErrors && !selected && (
            <div className="mt-1.5 space-y-0.5">
              {validationErrors.map(({ name }) => (
                <div
                  key={name}
                  className="inline-flex items-center gap-1 rounded border border-amber-500/25 bg-amber-500/10 px-1.5 py-0.5 text-[10px] text-amber-300"
                >
                  <span className="font-medium">
                    {name
                      .split("_")
                      .map((p) => p.charAt(0).toUpperCase() + p.slice(1))
                      .join(" ")}
                  </span>
                  <span className="opacity-60">required</span>
                </div>
              ))}
            </div>
          )}

          {/* Existing EC2 / Ansible badges */}
          {isEc2Node && hasAnsiblePlaybook && (
            <div className="mt-1 inline-flex items-center rounded-full border border-emerald-400/30 bg-emerald-500/10 px-2 py-0.5 text-[10px] font-medium text-emerald-200">
              Playbook uploaded
            </div>
          )}
          {hasQueuedAnsibleJob && (
            <div className="mt-1 inline-flex items-center rounded-full border border-cyan-400/30 bg-cyan-500/10 px-2 py-0.5 text-[10px] font-medium text-cyan-200">
              {data.lastAnsibleJobStatus ?? "QUEUED"}
            </div>
          )}
          {isEc2Node && data.ansiblePlaybookName && (
            <div className="mt-1 max-w-[160px] truncate text-[11px] text-slate-300">
              {data.ansiblePlaybookName}
            </div>
          )}
          {hasQueuedAnsibleJob && (
            <div className="mt-1 max-w-[160px] truncate text-[11px] text-slate-300">
              Job: {data.lastAnsibleJobId}
            </div>
          )}
        </div>
      </div>

      <Handle
        type="target"
        position={Position.Left}
        style={{
          width: 12, height: 12, borderRadius: 999,
          border: "2px solid rgba(100,180,255,0.5)",
          background: "rgba(30,35,45,0.95)",
          left: -6,
        }}
      />
      <Handle
        type="source"
        position={Position.Right}
        style={{
          width: 12, height: 12, borderRadius: 999,
          border: "2px solid rgba(100,180,255,0.5)",
          background: "rgba(30,35,45,0.95)",
          right: -6,
        }}
      />
    </div>
  );
}