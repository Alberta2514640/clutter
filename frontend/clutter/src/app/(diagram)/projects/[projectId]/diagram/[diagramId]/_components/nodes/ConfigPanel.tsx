"use client";

import { useDiagramActions, useDiagramState } from "@/lib/stores/diagramStore";
import { Settings, X } from "lucide-react";
import { useMemo } from "react";

export default function ConfigPanel() {
  const { nodes } = useDiagramState();
  const { updateNode } = useDiagramActions();

  const selectedNode = useMemo(() => nodes.find((n) => n.selected), [nodes]);
  const showContent = !!selectedNode;

  const handleLabelChange = (newLabel: string) => {
    if (!selectedNode) return;
    updateNode(selectedNode.id, {
      ...selectedNode,
      data: { ...selectedNode.data, label: newLabel },
    });
  };

  const handleClose = () => {
    if (!selectedNode) return;
    updateNode(selectedNode.id, { ...selectedNode, selected: false });
  };

  return (
    <aside
      className={[
        "relative h-full shrink-0 border-l border-slate-800 bg-slate-950/70 backdrop-blur",
        "transition-[width] duration-200",
        showContent ? "w-[200px]" : "w-[5px]",
      ].join(" ")}
    >
      {/* collapsed = tiny rail only */}
      {!showContent ? null : (
        <>
          {/* Header like Palette */}
          <div className="flex h-14 items-center justify-between gap-2 border-b border-slate-800 px-3">
            <div className="flex items-center gap-2 min-w-0">
              <div className="grid h-8 w-8 place-items-center rounded-lg border border-slate-800 bg-slate-900">
                <Settings className="h-4 w-4 text-slate-300" />
              </div>
              <div className="min-w-0">
                <div className="truncate text-sm font-semibold text-slate-100">
                  Config
                </div>
                <div className="truncate text-[11px] text-slate-400">
                  Node Inspector
                </div>
              </div>
            </div>

            <button
              onClick={handleClose}
              className="grid h-8 w-8 place-items-center rounded-md border border-slate-800 bg-slate-900 hover:bg-slate-800 transition"
              title="Deselect node"
            >
              <X className="h-4 w-4 text-slate-300" />
            </button>
          </div>

          {/* Body */}
          <div className="h-[calc(100%-3.5rem)] overflow-y-auto p-4">
            <div className="space-y-4">
              {/* Node Type Badge */}
              <div className="flex items-center gap-3 rounded-lg border border-slate-800 bg-slate-900/40 p-3">
                <div className="grid h-10 w-10 shrink-0 place-items-center rounded-md border border-slate-800 bg-slate-950 text-sm font-bold text-gray-300">
                  {selectedNode!.data.img}
                </div>
                <div className="min-w-0">
                  <div className="text-xs text-gray-400">Type</div>
                  <div className="truncate text-sm font-medium text-white">
                    {selectedNode!.type === "awsService" ? "AWS Service" : selectedNode!.type}
                  </div>
                </div>
              </div>

              {/* Label Input */}
              <div>
                <label className="mb-2 block text-xs font-semibold uppercase tracking-wider text-gray-500">
                  Label
                </label>
                <input
                  type="text"
                  value={selectedNode!.data.label}
                  onChange={(e) => handleLabelChange(e.target.value)}
                  className="w-full rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-2 text-sm text-white transition-colors focus:border-teal-500 focus:outline-none focus:ring-1 focus:ring-teal-500"
                />
              </div>

              {/* Position Info */}
              <div>
                <label className="mb-2 block text-xs font-semibold uppercase tracking-wider text-gray-500">
                  Position
                </label>
                <div className="grid grid-cols-2 gap-2">
                  <div className="rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-2">
                    <div className="text-xs text-gray-400">X</div>
                    <div className="text-sm font-medium text-white">
                      {Math.round(selectedNode!.position.x)}
                    </div>
                  </div>
                  <div className="rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-2">
                    <div className="text-xs text-gray-400">Y</div>
                    <div className="text-sm font-medium text-white">
                      {Math.round(selectedNode!.position.y)}
                    </div>
                  </div>
                </div>
              </div>

              {/* Node ID */}
              <div>
                <label className="mb-2 block text-xs font-semibold uppercase tracking-wider text-gray-500">
                  Node ID
                </label>
                <div className="overflow-hidden rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-2">
                  <div className="truncate text-xs text-gray-400">
                    {selectedNode!.id}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </>
      )}
    </aside>
  );
}