"use client";

import React, { useState, useRef, useCallback } from "react";

const GRID_SIZE = 24;
const NODE_WIDTH = 120;
const NODE_HEIGHT = 42;

type NodeType = "enclosure" | "shape" | "custom" | "dynamodb" | "s3" | "lambda" | "apigw";

interface NodeInstance {
  id: string;
  type: NodeType;
  x: number;
  y: number;
}

interface Edge {
  id: string;
  fromId: string;
  toId: string;
}

const paletteItems: { label: string; type: NodeType }[] = [
  { label: "Enclosure", type: "enclosure" },
  { label: "Shapes", type: "shape" },
  { label: "Custom", type: "custom" },
  { label: "DynamoDB", type: "dynamodb" },
  { label: "S3", type: "s3" },
  { label: "Lambda", type: "lambda" },
  { label: "API Gateway", type: "apigw" },
];

const CanvasPage: React.FC = () => {
  const [nodes, setNodes] = useState<NodeInstance[]>([]);
  const [edges, setEdges] = useState<Edge[]>([]);
  const [dragging, setDragging] = useState<{
    id: string;
    offsetX: number;
    offsetY: number;
  } | null>(null);
  const [connectingFrom, setConnectingFrom] = useState<string | null>(null);

  const canvasRef = useRef<HTMLDivElement | null>(null);

  // ---------- PALETTE → CANVAS DROP ----------
  const handleCanvasDrop = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    const type = e.dataTransfer.getData("application/x-node-type") as NodeType;
    if (!type || !canvasRef.current) return;

    const rect = canvasRef.current.getBoundingClientRect();
    const rawX = e.clientX - rect.left;
    const rawY = e.clientY - rect.top;

    const x = Math.round(rawX / GRID_SIZE) * GRID_SIZE;
    const y = Math.round(rawY / GRID_SIZE) * GRID_SIZE;

    setNodes((prev) => [
      ...prev,
      {
        id: crypto.randomUUID(),
        type,
        x,
        y,
      },
    ]);
  };

  const handleCanvasDragOver = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
  };

  // ---------- NODE DRAGGING ----------
  const startNodeDrag = (e: React.PointerEvent<HTMLDivElement>, id: string) => {
    e.preventDefault();
    e.stopPropagation();
    if (!canvasRef.current) return;

    const rect = canvasRef.current.getBoundingClientRect();
    const node = nodes.find((n) => n.id === id);
    if (!node) return;

    const offsetX = e.clientX - rect.left - node.x;
    const offsetY = e.clientY - rect.top - node.y;

    (e.target as HTMLElement).setPointerCapture(e.pointerId);
    setDragging({ id, offsetX, offsetY });
  };

  const handleCanvasPointerMove = (e: React.PointerEvent<HTMLDivElement>) => {
    if (!dragging || !canvasRef.current) return;

    const rect = canvasRef.current.getBoundingClientRect();
    const rawX = e.clientX - rect.left - dragging.offsetX;
    const rawY = e.clientY - rect.top - dragging.offsetY;

    const x = Math.round(rawX / GRID_SIZE) * GRID_SIZE;
    const y = Math.round(rawY / GRID_SIZE) * GRID_SIZE;

    setNodes((prev) => prev.map((n) => (n.id === dragging.id ? { ...n, x, y } : n)));
  };

  const stopCanvasDrag = () => {
    if (!dragging) return;
    setDragging(null);
  };

  // ---------- CONNECTIONS ----------
  const handleNodeClick = (nodeId: string) => {
    if (!connectingFrom) {
      setConnectingFrom(nodeId);
      return;
    }

    if (connectingFrom === nodeId) {
      // clicking again cancels
      setConnectingFrom(null);
      return;
    }

    // create edge
    setEdges((prev) => [...prev, { id: crypto.randomUUID(), fromId: connectingFrom, toId: nodeId }]);
    setConnectingFrom(null);
  };

  const handleCanvasClick = () => {
    // click on empty canvas cancels connection mode
    if (connectingFrom) setConnectingFrom(null);
  };

  const renderNodeLabel = useCallback((type: NodeType) => {
    switch (type) {
      case "dynamodb":
        return "DynamoDB";
      case "s3":
        return "S3";
      case "lambda":
        return "Lambda";
      case "apigw":
        return "API Gateway";
      case "enclosure":
        return "Enclosure";
      case "shape":
        return "Shape";
      case "custom":
        return "Custom";
      default:
        return type;
    }
  }, []);

  // compute center for connectors
  const getNodeCenter = (node: NodeInstance) => ({
    cx: node.x + NODE_WIDTH / 2,
    cy: node.y + NODE_HEIGHT / 2,
  });

  return (
    <div className="flex h-screen w-screen bg-black text-slate-900">
      {/* SIDEBAR */}
      <aside className="w-64 bg-white border-r border-slate-200 flex flex-col">
        <div className="px-6 py-4 border-b border-slate-200">
          <div className="flex items-center gap-2">
            <span className="text-2xl font-bold text-sky-500">Clutter</span>
            <div className="h-6 w-6 rounded-md bg-sky-500" />
          </div>
        </div>

        <div className="flex-1 overflow-y-auto px-4 py-4 space-y-6 text-sm">
          <PaletteSection title="General">
            {paletteItems
              .filter((i) => ["enclosure", "shape", "custom"].includes(i.type))
              .map((item) => (
                <PaletteItem key={item.type} item={item} />
              ))}
          </PaletteSection>

          <PaletteSection title="Storage">
            {paletteItems
              .filter((i) => ["dynamodb", "s3"].includes(i.type))
              .map((item) => (
                <PaletteItem key={item.type} item={item} />
              ))}
          </PaletteSection>

          <PaletteSection title="Compute">
            {paletteItems
              .filter((i) => i.type === "lambda")
              .map((item) => (
                <PaletteItem key={item.type} item={item} />
              ))}
          </PaletteSection>

          <PaletteSection title="Network">
            {paletteItems
              .filter((i) => i.type === "apigw")
              .map((item) => (
                <PaletteItem key={item.type} item={item} />
              ))}
          </PaletteSection>
        </div>
      </aside>

      {/* MAIN */}
      <main className="flex-1 flex flex-col">
        <header className="h-14 border-b border-slate-200 flex items-center justify-between px-6 bg-white">
          <div className="flex items-center gap-3">
            <span className="text-sm text-slate-500">Cloud Provider</span>
            <button className="px-3 py-1 rounded-full border text-sm bg-slate-50">AWS ▼</button>
          </div>

          <div className="flex items-center gap-3">
            <button className="h-8 w-8 rounded-full border" />
            <button className="px-3 py-1 rounded-full bg-sky-500 text-white text-sm">Help</button>
            <button className="h-9 w-9 rounded-full bg-slate-200" />
          </div>
        </header>

        <div className="flex-1 flex items-stretch justify-stretch p-4">
          <div
            ref={canvasRef}
            className="relative flex-1 rounded-2xl border border-slate-300 bg-slate-100 overflow-hidden"
            onDrop={handleCanvasDrop}
            onDragOver={handleCanvasDragOver}
            onPointerMove={handleCanvasPointerMove}
            onPointerUp={stopCanvasDrag}
            onPointerLeave={stopCanvasDrag}
            onClick={handleCanvasClick}>
            {/* dotted grid */}
            <div className="absolute inset-0 pointer-events-none bg-[radial-gradient(circle,_#d4d4d4_1px,_transparent_0)] bg-[length:24px_24px]" />

            {/* CONNECTION LAYER */}
            <svg className="absolute inset-0 pointer-events-none">
              <defs>
                <marker id="arrowhead" markerWidth="8" markerHeight="8" refX="4" refY="4" orient="auto">
                  <path d="M0,0 L8,4 L0,8 z" fill="#64748b" />
                </marker>
              </defs>
              {edges.map((edge) => {
                const from = nodes.find((n) => n.id === edge.fromId);
                const to = nodes.find((n) => n.id === edge.toId);
                if (!from || !to) return null;

                const { cx: x1, cy: y1 } = getNodeCenter(from);
                const { cx: x2, cy: y2 } = getNodeCenter(to);

                return <line key={edge.id} x1={x1} y1={y1} x2={x2} y2={y2} stroke="#64748b" strokeWidth={2} markerEnd="url(#arrowhead)" />;
              })}
            </svg>

            {/* NODES */}
            {nodes.map((node) => {
              const isSelected = connectingFrom === node.id;
              return (
                <div
                  key={node.id}
                  className="absolute z-10 cursor-move select-none"
                  style={{
                    transform: `translate(${node.x}px, ${node.y}px)`,
                    width: NODE_WIDTH,
                    height: NODE_HEIGHT,
                  }}
                  onPointerDown={(e) => startNodeDrag(e, node.id)}
                  onClick={(e) => {
                    e.stopPropagation();
                    handleNodeClick(node.id);
                  }}>
                  <div className={`w-full h-full px-3 py-2 rounded-md bg-white shadow-sm border flex items-center gap-2 text-xs ${isSelected ? "border-sky-500 ring-2 ring-sky-200" : "border-slate-300"}`}>
                    <NodeIcon type={node.type} />
                    <span className="font-medium text-slate-800">{renderNodeLabel(node.type)}</span>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </main>
    </div>
  );
};

export default CanvasPage;

/* ---------- SMALL COMPONENTS ---------- */

const PaletteSection: React.FC<{ title: string; children: React.ReactNode }> = ({ title, children }) => (
  <section>
    <div className="font-semibold text-xs uppercase text-slate-500 mb-2">{title}</div>
    <div className="space-y-2">{children}</div>
  </section>
);

const PaletteItem: React.FC<{
  item: { label: string; type: NodeType };
}> = ({ item }) => {
  const handleDragStart = (e: React.DragEvent<HTMLDivElement>) => {
    e.dataTransfer.setData("application/x-node-type", item.type);
    e.dataTransfer.effectAllowed = "copy";
  };

  return (
    <div draggable onDragStart={handleDragStart} className="flex items-center gap-2 px-2 py-1.5 rounded-md border border-dashed border-slate-300 hover:border-sky-400 hover:bg-sky-50 cursor-grab active:cursor-grabbing">
      <NodeIcon type={item.type} />
      <span className="text-xs text-slate-800">{item.label}</span>
    </div>
  );
};

// Simple inline icons; swap these for real AWS SVGs when you’re ready
const NodeIcon: React.FC<{ type: NodeType }> = ({ type }) => {
  const base = "flex items-center justify-center rounded-md text-[10px] font-bold h-6 w-6";

  switch (type) {
    case "dynamodb":
      return <div className={`${base} bg-indigo-100 text-indigo-700 border border-indigo-300`}>DB</div>;
    case "s3":
      return <div className={`${base} bg-emerald-100 text-emerald-700 border border-emerald-300`}>S3</div>;
    case "lambda":
      return <div className={`${base} bg-amber-100 text-amber-700 border border-amber-300`}>λ</div>;
    case "apigw":
      return <div className={`${base} bg-yellow-100 text-yellow-700 border border-yellow-300`}>API</div>;
    case "enclosure":
      return <div className={`${base} bg-slate-100 text-slate-700 border border-slate-300`}>ENC</div>;
    case "shape":
      return <div className={`${base} bg-slate-100 text-slate-700 border border-slate-300`}>SH</div>;
    case "custom":
      return <div className={`${base} bg-sky-100 text-sky-700 border border-sky-300`}>*</div>;
    default:
      return <div className={`${base} bg-slate-200`} />;
  }
};
