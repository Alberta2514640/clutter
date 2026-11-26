"use client";

import { BaseEdge, getBezierPath, Position, useInternalNode, type EdgeProps, type InternalNode } from "@xyflow/react";
import { ArrowRight } from "lucide-react";
import { useState } from "react";
import { createPortal } from "react-dom";
import { useCanvas, type EdgeCard } from "./canvasContext";
import { kindIconClass } from "@/lib/features/catalog/display";
import KindIcon from "./KindIcon";

export const FLOATING_EDGE_TYPE = "floating";

type Box = { x: number; y: number; w: number; h: number; cx: number; cy: number };

function boxOf(n: InternalNode): Box | null {
  const w = n.measured.width;
  const h = n.measured.height;
  if (!w || !h) return null;
  const { x, y } = n.internals.positionAbsolute;
  return { x, y, w, h, cx: x + w / 2, cy: y + h / 2 };
}

// The two sides of the blocks that face each other: left and right when the
// blocks are further apart across than up and down, top and bottom otherwise.
function facingSides(s: Box, t: Box) {
  const dx = t.cx - s.cx;
  const dy = t.cy - s.cy;
  const gapAcross = Math.abs(dx) - (s.w + t.w) / 2;
  const gapDown = Math.abs(dy) - (s.h + t.h) / 2;
  if (gapAcross >= gapDown) {
    const right = dx >= 0;
    return {
      sx: right ? s.x + s.w : s.x,
      sy: s.cy,
      sourcePosition: right ? Position.Right : Position.Left,
      tx: right ? t.x : t.x + t.w,
      ty: t.cy,
      targetPosition: right ? Position.Left : Position.Right,
    };
  }
  const down = dy >= 0;
  return {
    sx: s.cx,
    sy: down ? s.y + s.h : s.y,
    sourcePosition: down ? Position.Bottom : Position.Top,
    tx: t.cx,
    ty: down ? t.y : t.y + t.h,
    targetPosition: down ? Position.Top : Position.Bottom,
  };
}

// An edge drawn between the sides of its two blocks that face each other,
// whatever handle it was drawn from, so it never loops around a block. The
// saved document keeps only source and target, so nothing about it changes.
// Hovering it shows a small card: both ends, what the line means and why.
export default function FloatingEdge({ id, source, target, markerEnd, style, interactionWidth, selected }: EdgeProps) {
  const sourceNode = useInternalNode(source);
  const targetNode = useInternalNode(target);
  const canvas = useCanvas();
  const [hover, setHover] = useState<{ x: number; y: number } | null>(null);
  const s = sourceNode ? boxOf(sourceNode) : null;
  const t = targetNode ? boxOf(targetNode) : null;
  if (!s || !t) return null;

  const { sx, sy, sourcePosition, tx, ty, targetPosition } = facingSides(s, t);
  const [path] = getBezierPath({ sourceX: sx, sourceY: sy, sourcePosition, targetX: tx, targetY: ty, targetPosition });
  const card = hover ? canvas.edgeCard(source, target) : null;
  return (
    <g
      data-testid="block-edge"
      className="animate-fade-in"
      onMouseEnter={(e) => setHover({ x: e.clientX, y: e.clientY })}
      onMouseMove={(e) => setHover({ x: e.clientX, y: e.clientY })}
      onMouseLeave={() => setHover(null)}>
      <BaseEdge
        id={id}
        path={path}
        markerEnd={markerEnd}
        style={hover || selected ? { ...style, stroke: "var(--primary)", strokeWidth: 2 } : style}
        interactionWidth={interactionWidth}
      />
      {card && hover && typeof document !== "undefined" && createPortal(<LinkCardView card={card} at={hover} />, document.body)}
    </g>
  );
}

// The hover card, beside the pointer, kept inside the window.
function LinkCardView({ card, at }: { card: EdgeCard; at: { x: number; y: number } }) {
  const left = Math.min(at.x + 14, window.innerWidth - 300);
  const top = Math.min(at.y + 14, window.innerHeight - 140);
  return (
    <div
      role="tooltip"
      data-testid="link-card"
      style={{ left, top }}
      className="surface-raised text-popover-foreground pointer-events-none fixed z-[1000] w-72 rounded-xl border px-3 py-2.5 text-xs animate-fade-in">
      <LinkCardBody card={card} />
    </div>
  );
}

// LinkCardBody is a line's card: the two ends with their icons, then what the
// line means and why it helps. "How linking works" shows the same body.
export function LinkCardBody({ card }: { card: EdgeCard }) {
  return (
    <>
      <div className="flex items-center gap-1.5 font-medium">
        <KindIcon kind={card.source} className={`${kindIconClass(card.source)} size-4 shrink-0`} />
        <span className="truncate">{card.sourceName}</span>
        <ArrowRight className="text-muted-foreground size-3.5 shrink-0" aria-label="to" />
        <KindIcon kind={card.target} className={`${kindIconClass(card.target)} size-4 shrink-0`} />
        <span className="truncate">{card.targetName}</span>
      </div>
      <p className="mt-1.5 leading-snug">{card.meaning}</p>
      {card.why && <p className="text-muted-foreground mt-1 leading-snug">{card.why}</p>}
    </>
  );
}
