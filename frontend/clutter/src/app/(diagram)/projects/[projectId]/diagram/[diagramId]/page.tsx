"use client";

import type { Modifier } from "@dnd-kit/core";
import { DndContext, DragCancelEvent, DragEndEvent, DragOverlay, DragStartEvent, PointerSensor, useSensor, useSensors, } from "@dnd-kit/core";
import { useCallback, useEffect, useRef, useState } from "react";

import { Canvas } from "./_components/canvas/Canvas";
import { Palette } from "./_components/palette/Palette";
import { NodeIcon } from "./_components/shared/NodeIcon";
import { clamp, NODE_HEIGHT, NODE_WIDTH, renderNodeLabel, snap, uid } from "./_components/shared/constants";
import type { ActiveDrag, DragData, Edge, NodeInstance, NodeType } from "./_components/types";

export default function DiagramPage() {
  const [mounted, setMounted] = useState(false);
  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => setMounted(true), []);

  const [nodes, setNodes] = useState<NodeInstance[]>([]);
  const [edges, setEdges] = useState<Edge[]>([]);
  const [connectingFrom, setConnectingFrom] = useState<string | null>(null);
  const [activeDrag, setActiveDrag] = useState<ActiveDrag>(null);

  const canvasRef = useRef<HTMLDivElement | null>(null);
  const lastPointerRef = useRef<{ x: number; y: number } | null>(null);

  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 4 } }));

  const onWindowPointerMove = useCallback((e: PointerEvent) => {
    lastPointerRef.current = { x: e.clientX, y: e.clientY };
  }, []);

  useEffect(() => {
    console.log("[Diagram state]", {
      nodes,
      edges,
      connectingFrom,
      activeDrag,
      canvasEl: canvasRef.current,
      lastPointer: lastPointerRef.current,
      sensors,
    });
  }, [nodes, edges, connectingFrom, activeDrag, sensors]);

  const startTrackingPointer = useCallback(() => {
    window.addEventListener("pointermove", onWindowPointerMove, { passive: true });
  }, [onWindowPointerMove]);

  const stopTrackingPointer = useCallback(() => {
    window.removeEventListener("pointermove", onWindowPointerMove);
  }, [onWindowPointerMove]);

  useEffect(() => {
    return () => stopTrackingPointer();
  }, [stopTrackingPointer]);

  const handleNodeClick = (nodeId: string) => {
    if (!connectingFrom) return void setConnectingFrom(nodeId);
    if (connectingFrom === nodeId) return void setConnectingFrom(null);
    setEdges((prev) => [...prev, { id: uid(), fromId: connectingFrom, toId: nodeId }]);
    setConnectingFrom(null);
  };

  const handleCanvasClick = () => {
    if (connectingFrom) setConnectingFrom(null);
  };

  const paletteOverlayModifier: Modifier = ({ transform }) => ({
    ...transform,
    x: transform.x - NODE_WIDTH / 2,
    y: transform.y - NODE_HEIGHT / 2,
  });

  const handleDragStart = (event: DragStartEvent) => {
    const data = event.active.data.current as unknown;
    if (!data || typeof data !== "object") return;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const ae = event.activatorEvent as any;
    if (ae && typeof ae.clientX === "number" && typeof ae.clientY === "number") {
      lastPointerRef.current = { x: ae.clientX, y: ae.clientY };
    }

    const d = data as { kind?: string; nodeType?: NodeType };

    if (d.kind === "palette" && d.nodeType) {
      setActiveDrag({ kind: "palette", type: d.nodeType });
      startTrackingPointer();
      return;
    }

    if (d.kind === "node") {
      setActiveDrag({ kind: "node", id: String(event.active.id) });
    }
  };

  const handleDragEnd = (event: DragEndEvent) => {
    const data = event.active.data.current as DragData | undefined;
    setActiveDrag(null);

    if (!data) return;

    // --- PALETTE: pointer-based create ---
    if (data.kind === "palette") {
      stopTrackingPointer();

      if (!canvasRef.current) return;
      const pt = lastPointerRef.current;
      if (!pt) return;

      const rect = canvasRef.current.getBoundingClientRect();
      const inside = pt.x >= rect.left && pt.x <= rect.right && pt.y >= rect.top && pt.y <= rect.bottom;
      if (!inside) return;

      const px = pt.x - rect.left;
      const py = pt.y - rect.top;

      const maxX = Math.max(0, rect.width - NODE_WIDTH);
      const maxY = Math.max(0, rect.height - NODE_HEIGHT);

      const x = clamp(snap(px - NODE_WIDTH / 2), 0, snap(maxX));
      const y = clamp(snap(py - NODE_HEIGHT / 2), 0, snap(maxY));

      setNodes((prev) => [...prev, { id: uid(), type: data.nodeType, x, y }]);
      return;
    }

    // --- NODE: delta-based move (the “perfect drag” part) ---
    if (data.kind === "node") {
      const id = String(event.active.id);
      const dx = event.delta.x;
      const dy = event.delta.y;

      const rect = canvasRef.current?.getBoundingClientRect();
      const maxX = rect ? Math.max(0, rect.width - NODE_WIDTH) : Number.POSITIVE_INFINITY;
      const maxY = rect ? Math.max(0, rect.height - NODE_HEIGHT) : Number.POSITIVE_INFINITY;

      setNodes((prev) =>
        prev.map((n) => {
          if (n.id !== id) return n;

          const nx = snap(n.x + dx);
          const ny = snap(n.y + dy);

          return {
            ...n,
            x: clamp(nx, 0, isFinite(maxX) ? snap(maxX) : nx),
            y: clamp(ny, 0, isFinite(maxY) ? snap(maxY) : ny),
          };
        })
      );
    }
  };


  const handleDragCancel = (_event: DragCancelEvent) => {
    stopTrackingPointer();
    setActiveDrag(null);
  };

  return (
    <div className="w-full px-6 py-6">
      <div className="rounded-2xl border border-white/10 bg-white/5 backdrop-blur p-6">
        {!mounted ? (
          <div className="flex gap-6">
            <aside className="w-[280px] shrink-0">
              <Palette mounted={false} />
            </aside>

            <section className="flex-1 min-w-0">
              <div className="relative w-full min-h-[72vh] rounded-2xl border border-white/10 bg-white/5" />
            </section>
          </div>
        ) : (
          <DndContext
            sensors={sensors}
            onDragStart={handleDragStart}
            onDragEnd={handleDragEnd}
            onDragCancel={handleDragCancel}
          >
            <div className="flex gap-6">
              <Palette mounted />
            
              <section className="flex-1 min-w-0">
                <Canvas
                  canvasRef={canvasRef}
                  nodes={nodes}
                  edges={edges}
                  connectingFrom={connectingFrom}
                  onNodeClick={handleNodeClick}
                  onCanvasClick={handleCanvasClick}
                />
              </section>
            </div>

            <DragOverlay
              dropAnimation={null}
              modifiers={activeDrag?.kind === "palette" ? [paletteOverlayModifier] : undefined}
            >
              {activeDrag?.kind === "palette" ? (
                <div
                  style={{ width: NODE_WIDTH, height: NODE_HEIGHT }}
                  className="rounded-xl border border-white/10 bg-slate-900/80 backdrop-blur px-3 py-2 text-xs text-slate-100 shadow-2xl flex items-center gap-2"
                >
                  <NodeIcon type={activeDrag.type} />
                  <span className="font-medium">{renderNodeLabel(activeDrag.type)}</span>
                </div>
              ) : null}
            </DragOverlay>
          </DndContext>
        )}
      </div>
    </div>
  );
}