"use client";

import { fieldShown } from "@/lib/features/catalog/forms";
import { BOX, type BlockNode as BlockNodeType } from "@/lib/features/diagram/blocksDocument";
import { Handle, NodeResizer, Position, type NodeProps } from "@xyflow/react";
import { useCanvas } from "./canvasContext";
import HelpTip from "./HelpTip";

// A network on B's canvas: a resizable dashed box labelled "name · CIDR". A
// VM inside it runs in that network (ADR-025). While a VM is dragged over it,
// it lights up and says where the VM will run. Its on and off settings sit on
// its lower edge, each with its help behind a "?".
export default function NetworkBox({ id, data, selected }: NodeProps<BlockNodeType>) {
  const canvas = useCanvas();
  const fields = canvas.fieldsOf(data.kind);
  const cidrField = fields.find((f) => f.name === "cidr");
  const cidr = typeof data.fields.cidr === "string" ? data.fields.cidr : typeof cidrField?.default === "string" ? cidrField.default : "";
  const switches = fields.filter((f) => f.type === "bool" && fieldShown(f, { hasPrivateVM: canvas.hasPrivateVM(id) }));
  const dropping = canvas.dropTarget === id;
  const problems = canvas.problems(id);
  const name = data.name || "(no name)";

  return (
    <div
      data-testid="network-box"
      data-drop={dropping || undefined}
      className={`relative h-full w-full rounded-2xl border-[1.5px] transition-colors ${
        dropping
          ? "border-solid border-teal-300 bg-teal-400/[0.07] shadow-[0_0_0_4px_rgba(45,212,191,0.15)]"
          : selected
            ? "border-dashed border-teal-400 bg-teal-500/[0.05]"
            : "border-dashed border-teal-700/80 bg-teal-500/[0.035]"
      }`}>
      <NodeResizer
        isVisible={selected}
        minWidth={BOX.minWidth}
        minHeight={BOX.minHeight}
        color="#2dd4bf"
        handleClassName="!h-2.5 !w-2.5 !rounded-sm !border-slate-950"
      />
      {/* Lines never end on a box: containment is the VM-to-Network link. This
          hidden target only draws a VM's second network link from an older diagram. */}
      <Handle type="target" position={Position.Left} isConnectable={false} className="!pointer-events-none !opacity-0" />
      <div className="absolute -top-2.5 left-3 flex max-w-[calc(100%-1.5rem)] items-center gap-1 bg-[#020617] px-2 text-xs leading-5">
        <span className="truncate font-medium text-teal-200">{name}</span>
        {cidr && <span className="shrink-0 text-slate-400">· {cidr}</span>}
        {problems.length > 0 && (
          <span data-testid="problem-badge" title={problems.join(" ")} className="ml-1 flex h-4 w-4 items-center justify-center rounded-full bg-amber-400 text-[11px] font-bold text-slate-950">
            !<span className="sr-only">: {problems.join(" ")}</span>
          </span>
        )}
      </div>
      {dropping && (
        <div className="pointer-events-none absolute bottom-7 left-1/2 -translate-x-1/2 whitespace-nowrap rounded-full border border-teal-700 bg-teal-950/90 px-3 py-0.5 text-[11px] text-teal-200">
          Drop to run it in {name}
        </div>
      )}
      {switches.length > 0 && (
        <div className="absolute -bottom-2.5 right-3 flex items-center gap-3 bg-[#020617] px-2">
          {switches.map((f) => {
            const on = data.fields[f.name] === undefined ? f.default === true : data.fields[f.name] === true;
            return (
              <span key={f.name} className="flex items-center gap-1.5 text-[11px] leading-5 text-slate-400">
                <button
                  type="button"
                  role="switch"
                  aria-checked={on}
                  aria-label={`${f.title} in ${name}`}
                  onClick={() => canvas.setField(id, f.name, !on)}
                  className={`nodrag nopan relative h-3.5 w-6 rounded-full transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-teal-300 ${
                    on ? "bg-teal-500" : "bg-slate-700"
                  }`}>
                  <span className={`absolute top-0.5 h-2.5 w-2.5 rounded-full bg-white transition-all ${on ? "left-3" : "left-0.5"}`} />
                </button>
                <span className="whitespace-nowrap">{f.title}</span>
                {f.help && <HelpTip text={f.help} label={`About ${f.title}`} align="end" />}
              </span>
            );
          })}
        </div>
      )}
    </div>
  );
}
