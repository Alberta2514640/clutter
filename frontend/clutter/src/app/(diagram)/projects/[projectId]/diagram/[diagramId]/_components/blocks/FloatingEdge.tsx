"use client";

import { BaseEdge, getBezierPath, Position, useInternalNode, type EdgeProps, type InternalNode } from "@xyflow/react";

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
export default function FloatingEdge({ id, source, target, markerEnd, style, interactionWidth }: EdgeProps) {
  const sourceNode = useInternalNode(source);
  const targetNode = useInternalNode(target);
  const s = sourceNode ? boxOf(sourceNode) : null;
  const t = targetNode ? boxOf(targetNode) : null;
  if (!s || !t) return null;

  const { sx, sy, sourcePosition, tx, ty, targetPosition } = facingSides(s, t);
  const [path] = getBezierPath({ sourceX: sx, sourceY: sy, sourcePosition, targetX: tx, targetY: ty, targetPosition });
  return <BaseEdge id={id} path={path} markerEnd={markerEnd} style={style} interactionWidth={interactionWidth} />;
}
