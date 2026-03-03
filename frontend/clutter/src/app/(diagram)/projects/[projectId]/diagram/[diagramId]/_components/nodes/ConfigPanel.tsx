"use client";

import { Settings, Trash2, X } from "lucide-react";
import { useCallback, useMemo } from "react";
import Image from "next/image";

import type { DiagramNode } from "@/lib/features/diagram/types";
import { useDiagramEditor, useDiagramEditorActions } from "@/lib/features/diagram/uiStore";

export default function ConfigPanel({ diagramId }: { diagramId: string }) {
  const editor = useDiagramEditor(diagramId);
  const { setNodes, setEdges } = useDiagramEditorActions();

  const nodes = useMemo(() => editor?.nodes ?? [], [editor?.nodes]);
  const edges = useMemo(() => editor?.edges ?? [], [editor?.edges]);

  const selectedNode = useMemo(() => nodes.find((n) => n.selected), [nodes]);
  const showContent = !!selectedNode;

  const patchSelectedNode = useCallback(
    (patch: (node: DiagramNode) => DiagramNode) => {
      if (!selectedNode) return;

      const next = nodes.map((n) => (n.id === selectedNode.id ? patch(n) : n));
      setNodes(diagramId, next);
    },
    [diagramId, nodes, selectedNode, setNodes],
  );

  const handleLabelChange = (newLabel: string) => {
    patchSelectedNode((n) => ({
      ...n,
      data: { ...n.data, label: newLabel },
    }));
  };

  const handleClose = () => {
    patchSelectedNode((n) => ({ ...n, selected: false }));
  };

  const handleDeleteSelected = useCallback(() => {
    if (!selectedNode) return;

    const nodeId = selectedNode.id;

    const nextNodes = nodes.filter((n) => n.id !== nodeId);
    const nextEdges = edges.filter((e) => e.source !== nodeId && e.target !== nodeId);

    setNodes(diagramId, nextNodes);
    setEdges(diagramId, nextEdges);
  }, [diagramId, edges, nodes, selectedNode, setEdges, setNodes]);

  return (
    <aside className={["relative h-full shrink-0 border-l border-slate-800 bg-slate-950/70 backdrop-blur", "transition-[width] duration-200", showContent ? "w-[250px]" : "w-[5px]"].join(" ")}>
      {!showContent ? null : (
        <>
          {/* Header */}
          <div className="flex h-14 items-center justify-between gap-2 border-b border-slate-800 px-3">
            <div className="flex items-center gap-2 min-w-0">
              <div className="grid h-8 w-8 place-items-center rounded-lg border border-slate-800 bg-slate-900">
                <Settings className="h-4 w-4 text-slate-300" />
              </div>
              <div className="min-w-0">
                <div className="truncate text-sm font-semibold text-slate-100">Config</div>
                <div className="truncate text-[11px] text-slate-400">Node Inspector</div>
              </div>
            </div>

            <button onClick={handleClose} className="grid h-8 w-8 place-items-center rounded-md border border-slate-800 bg-slate-900 hover:bg-slate-800 transition" title="Deselect node" type="button">
              <X className="h-4 w-4 text-slate-300" />
            </button>
          </div>

          {/* Body */}
          <div className="h-[calc(100%-3.5rem)] overflow-y-auto p-4">
            <div className="space-y-4">
              {/* Node Type Badge */}
              <div className="flex items-center gap-3 rounded-lg border border-slate-800 bg-slate-900/40 p-3">
                <div className="grid h-10 w-10 shrink-0 place-items-center rounded-md border border-slate-800 bg-slate-950">
                  <Image src={selectedNode!.data.img as string} alt="" width={24} height={24} unoptimized />
                </div>
                <div className="min-w-0">
                  <div className="text-xs text-gray-400">Type</div>
                  <div className="truncate text-sm font-medium text-white">{selectedNode!.type === "awsService" ? "AWS Service" : selectedNode!.type}</div>
                </div>
              </div>

              {/* Label Input */}
              <div>
                <label className="mb-2 block text-xs font-semibold uppercase tracking-wider text-gray-500">Label</label>
                <input
                  type="text"
                  value={selectedNode!.data.label}
                  onChange={(e) => handleLabelChange(e.target.value)}
                  className="w-full rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-2 text-sm text-white transition-colors focus:border-teal-500 focus:outline-none focus:ring-1 focus:ring-teal-500"
                />
              </div>

              {/* Position Info */}
              <div>
                <label className="mb-2 block text-xs font-semibold uppercase tracking-wider text-gray-500">Position</label>
                <div className="grid grid-cols-2 gap-2">
                  <div className="rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-2">
                    <div className="text-xs text-gray-400">X</div>
                    <div className="text-sm font-medium text-white">{Math.round(selectedNode!.position.x)}</div>
                  </div>
                  <div className="rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-2">
                    <div className="text-xs text-gray-400">Y</div>
                    <div className="text-sm font-medium text-white">{Math.round(selectedNode!.position.y)}</div>
                  </div>
                </div>
              </div>

              {/* Danger Zone */}
              <div className="pt-2">
                <div className="mb-2 text-[11px] font-semibold uppercase tracking-wider text-gray-500">Danger zone</div>

                <button
                  type="button"
                  onClick={handleDeleteSelected}
                  className={[
                    "flex w-full items-center justify-center gap-2 rounded-lg border px-3 py-2 text-sm font-medium transition",
                    "border-red-900/40 bg-red-950/40 text-red-200",
                    "hover:bg-red-950/60 hover:border-red-800/60",
                    "focus:outline-none focus:ring-1 focus:ring-red-500/60",
                  ].join(" ")}
                  title="Delete selected node">
                  <Trash2 className="h-4 w-4" />
                  Delete node
                </button>

                <p className="mt-2 text-xs text-slate-400">This will also remove any connections to this node.</p>
              </div>
            </div>
          </div>
        </>
      )}
    </aside>
  );
}
