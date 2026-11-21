"use client";

import { fieldShown } from "@/lib/features/catalog/forms";
import { BOX, type BlockNode as BlockNodeType } from "@/lib/features/diagram/blocksDocument";
import { cn } from "@/lib/utils";
import { Handle, NodeResizer, Position, type NodeProps } from "@xyflow/react";
import { useCanvas } from "./canvasContext";
import HelpTip from "./HelpTip";

// A network on B's canvas: a resizable dashed box labelled "name · CIDR". A
// VM inside it runs in that network (ADR-025). While a VM is dragged over it,
// it lights up and says where the VM will run. Its on and off settings sit on
// its lower edge, each with its help behind a "?". The label and the
// switches sit on the canvas colour, over the box's border.
export default function NetworkBox({ id, data, selected }: NodeProps<BlockNodeType>) {
  const canvas = useCanvas();
  const fields = canvas.fieldsOf(data.kind);
  const cidrField = fields.find((f) => f.name === "cidr");
  const cidr = typeof data.fields.cidr === "string" ? data.fields.cidr : typeof cidrField?.default === "string" ? cidrField.default : "";
  const switches = fields.filter((f) => f.type === "bool" && fieldShown(f, { hasVM: canvas.hasVM(id) }));
  const dropping = canvas.dropTarget === id;
  const problems = canvas.problems(id);
  const live = canvas.live(id);
  const notBuilt = canvas.notBuilt(id);
  const name = data.name || "(no name)";

  return (
    <div
      data-testid="network-box"
      data-drop={dropping || undefined}
      data-live={live ? true : undefined}
      className={cn(
        "relative h-full w-full rounded-2xl border-[1.5px] transition-[background-color,border-color,box-shadow] animate-fade-in",
        dropping
          ? "border-primary bg-primary/10 ring-primary/20 border-solid ring-4"
          : selected
            ? "border-tint-violet-fg border-dashed"
            : "border-tint-violet-fg/45 border-dashed",
      )}>
      <NodeResizer
        isVisible={selected && canvas.editable}
        minWidth={BOX.minWidth}
        minHeight={BOX.minHeight}
        color="var(--primary)"
        handleClassName="!size-2.5 !rounded-[2px] !border-background !bg-background !border-primary"
      />
      {/* Lines never end on a box: containment is the VM-to-Network link. This
          hidden target only draws a VM's second network link from an older diagram. */}
      <Handle type="target" position={Position.Left} isConnectable={false} className="!pointer-events-none !opacity-0" />
      <div className="bg-background absolute -top-3 left-3 z-10 flex max-w-[calc(100%-1.5rem)] items-center gap-1 rounded-md px-2 text-xs leading-6">
        {/* The name keeps its room first. The live marker says "Live", as a
            block's does, so it never cuts a word; where it runs is its title
            and is read out. */}
        <span className="text-foreground max-w-48 shrink-0 truncate font-semibold">{name}</span>
        {cidr && <span className="text-muted-foreground shrink-0 font-mono text-[11px]">· {cidr}</span>}
        {live && (
          <span data-testid="live-marker" title={live} className="text-success ml-1 inline-flex shrink-0 items-center gap-1 text-[10px]">
            <span aria-hidden className="bg-success-solid size-1.5 shrink-0 rounded-full" />
            Live
            <span className="sr-only">: {live}</span>
          </span>
        )}
        {notBuilt && (
          <span data-testid="not-built-marker" className="text-muted-foreground ml-1 shrink-0 text-[10px] whitespace-nowrap">
            Not deployed yet
          </span>
        )}
        {problems.length > 0 && (
          <span
            data-testid="problem-badge"
            title={problems.join(" ")}
            className="bg-warning text-warning-foreground ml-1 flex size-4 items-center justify-center rounded-full text-[11px] font-bold">
            !<span className="sr-only">: {problems.join(" ")}</span>
          </span>
        )}
      </div>
      {dropping && (
        <div className="border-primary/40 bg-background text-link pointer-events-none absolute bottom-7 left-1/2 -translate-x-1/2 rounded-full border px-3 py-0.5 text-[11px] font-medium whitespace-nowrap shadow-sm">
          Drop to run it in {name}
        </div>
      )}
      {switches.length > 0 && (
        <div className="bg-background absolute -bottom-3 right-3 z-10 flex items-center gap-3 rounded-md px-2">
          {switches.map((f) => {
            const on = data.fields[f.name] === undefined ? f.default === true : data.fields[f.name] === true;
            return (
              <span key={f.name} className="text-muted-foreground flex items-center gap-1.5 text-[11px] leading-6">
                {/* A 28x24 target (WCAG 2.5.8) around the 28x16 track; the
                    negative margin keeps the row's height. */}
                <button
                  type="button"
                  role="switch"
                  aria-checked={on}
                  aria-label={`${f.title} in ${name}`}
                  disabled={!canvas.editable}
                  onClick={() => canvas.setField(id, f.name, !on)}
                  className="nodrag nopan group/switch -my-1 flex h-6 w-7 items-center outline-none disabled:opacity-60">
                  <span
                    aria-hidden
                    className={cn(
                      "group-focus-visible/switch:ring-ring/50 relative h-4 w-7 rounded-full transition-colors group-focus-visible/switch:ring-[3px]",
                      on ? "bg-primary" : "bg-input",
                    )}>
                    <span className={cn("bg-background absolute top-0.5 left-0.5 size-3 rounded-full shadow-sm transition-transform", on && "translate-x-3")} />
                  </span>
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
