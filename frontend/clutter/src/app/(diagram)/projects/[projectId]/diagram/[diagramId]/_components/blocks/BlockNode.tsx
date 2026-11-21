"use client";

import CloudLogo from "@/components/common/CloudLogo";
import { kindBadgeClass } from "@/lib/features/catalog/display";
import { VM_KIND, type BlockNode as BlockNodeType } from "@/lib/features/diagram/blocksDocument";
import { cn } from "@/lib/utils";
import { Handle, Position, useConnection, type NodeProps } from "@xyflow/react";
import { ShieldAlert } from "lucide-react";
import { useId } from "react";
import { useCanvas } from "./canvasContext";
import ServiceIcon from "./ServiceIcon";

// A selected block gets a plain ring. No corner squares: a block has a fixed
// size, and squares at the corners read as resize handles. Only the network
// box resizes, and NetworkBox's NodeResizer draws its real handles.
function SelectionRing() {
  return <span aria-hidden className="border-primary pointer-events-none absolute -inset-1 rounded-[10px] border-2" />;
}

// One block on B's canvas. A line starts at the dot on its right. While a
// line is drawn the whole card is its target: a block the line may reach
// gets a ring, the others dim (canvasContext.linkState). A new block pops in.
export default function BlockNode({ id, data, selected }: NodeProps<BlockNodeType>) {
  const canvas = useCanvas();
  const look = canvas.look({ id, kind: data.kind });
  const connecting = useConnection((c) => c.inProgress);
  const link = connecting ? canvas.linkState(id) : null;
  const problems = canvas.problems(id);
  const warning = canvas.warning(id);
  const live = canvas.live(id);
  const notBuilt = canvas.notBuilt(id);
  const hint = canvas.hint(data.kind);
  const publicVM = data.kind === VM_KIND && data.fields.public === true;
  const tipId = useId();

  return (
    <div
      data-testid="block-node"
      data-kind={data.kind}
      data-dimmed={look.dimmed || undefined}
      data-link={link ?? undefined}
      data-live={live ? true : undefined}
      title={look.statusDetail ?? undefined}
      className={cn(
        "surface-card text-card-foreground relative min-w-[160px] max-w-[260px] rounded-lg px-3 py-2 transition-[opacity,box-shadow] animate-pop-in",
        link === "valid" ? "border-primary ring-primary/25 ring-4" : look.dimmed && "border-dashed",
        link === "invalid" && "opacity-35",
      )}>
      {selected && <SelectionRing />}
      {/* The target: the whole card, and only while a line is drawn, so it never gets in the way of a drag. */}
      <Handle
        type="target"
        position={Position.Left}
        isConnectableStart={false}
        className={cn("!absolute !inset-0 !h-full !w-full !transform-none !rounded-lg !border-0 !bg-transparent !opacity-0", !connecting && "!pointer-events-none")}
      />
      <div className={cn("flex items-center gap-2", look.dimmed && "opacity-45")}>
        <span className={cn(kindBadgeClass(data.kind), "flex size-7 shrink-0 items-center justify-center rounded-md")}>
          <ServiceIcon src={look.iconSrc} kind={data.kind} className="size-4" />
        </span>
        <div className="min-w-0">
          <div className="truncate text-sm font-semibold">{data.name || "(no name)"}</div>
          {/* In a cloud's view, that cloud's small logo beside its service name,
              so every cloud reads the same way. Decorative: the view switch
              and the block's status name the cloud. */}
          <div className="text-muted-foreground flex min-w-0 items-center gap-1 text-xs">
            {/* An "off" logo's cross sits past its lower right corner; the
                margin keeps it clear of the service name. */}
            {look.viewCloud && (
              <CloudLogo
                connector={look.viewCloud.connector}
                shortName={look.viewCloud.shortName}
                src={look.viewCloud.logoSrc}
                size={14}
                off={look.dimmed}
                className={look.dimmed ? "mr-1.5" : undefined}
              />
            )}
            <span className="truncate">{look.subtitle}</span>
          </div>
        </div>
      </div>
      {hint && <p className="text-muted-foreground mt-1.5 text-[11px] leading-snug">{hint}</p>}
      {/* One quiet row at most: public, live or not deployed yet, open to
          everyone, the view's status, and in the blocks view the clouds it
          deploys to. */}
      {(publicVM || live || notBuilt || warning || look.status || look.badges.length > 0) && (
        <div className="mt-1.5 flex flex-wrap items-center gap-1">
          {publicVM && <span className="border-border text-muted-foreground rounded-full border px-1.5 text-[10px] leading-4">public</span>}
          {live && (
            <span data-testid="live-marker" title={live} className="text-success inline-flex items-center gap-1 px-0.5 text-[10px] leading-4">
              <span aria-hidden className="bg-success-solid size-1.5 rounded-full" />
              Live
              <span className="sr-only">: {live}</span>
            </span>
          )}
          {notBuilt && (
            <span data-testid="not-built-marker" className="text-muted-foreground inline-flex items-center gap-1 px-0.5 text-[10px] leading-4">
              <span aria-hidden className="border-muted-foreground size-1.5 rounded-full border" />
              Not deployed yet
            </span>
          )}
          {warning && (
            <span
              data-testid="open-warning"
              title={warning}
              className="border-warning/45 text-warning inline-flex items-center gap-1 rounded-full border px-1.5 text-[10px] leading-4">
              <ShieldAlert className="size-3" aria-hidden />
              Open to everyone
              <span className="sr-only">: {warning}</span>
            </span>
          )}
          {look.status && (
            <span data-testid="block-status" className="text-muted-foreground text-[10px] leading-4 font-medium">
              {look.status}
              {look.statusDetail && <span className="sr-only">: {look.statusDetail}</span>}
            </span>
          )}
          {look.badges.length > 0 && (
            <span className="ml-auto flex items-center gap-1">
              {look.badges.map((b) => (
                <CloudLogo key={b.cloud.connector} connector={b.cloud.connector} shortName={b.cloud.shortName} src={b.cloud.logoSrc} size={14} off={!b.on} label={b.detail} />
              ))}
            </span>
          )}
        </div>
      )}
      {problems.length > 0 && (
        <span className="group/problem absolute -top-3 -right-3 z-10">
          <button
            type="button"
            data-testid="problem-badge"
            aria-label={`${data.name || "This block"} has a problem`}
            aria-describedby={tipId}
            className="nodrag bg-warning text-warning-foreground focus-visible:ring-ring/50 flex size-6 items-center justify-center rounded-full text-xs leading-none font-bold shadow outline-none focus-visible:ring-[3px]">
            !
          </button>
          <span
            role="tooltip"
            id={tipId}
            className="surface-raised text-popover-foreground pointer-events-none invisible absolute top-0 left-6 w-64 space-y-1 rounded-lg border px-3 py-2 text-xs leading-snug opacity-0 transition-opacity group-focus-within/problem:visible group-focus-within/problem:opacity-100 group-hover/problem:visible group-hover/problem:opacity-100">
            {problems.map((p) => (
              <span key={p} className="block">
                {p}
              </span>
            ))}
          </span>
        </span>
      )}
      <Handle
        type="source"
        position={Position.Right}
        isConnectable={canvas.editable}
        className={cn("!border-background !bg-primary !size-3 !border-2", !canvas.editable && "!hidden")}
      />
    </div>
  );
}
