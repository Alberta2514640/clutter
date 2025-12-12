"use client";
import { useDiagramActions, useDiagramState } from "@/lib/stores/diagramStore";
import { X } from "lucide-react";

export default function ConfigPanel() {
  const { nodes } = useDiagramState();
  const { updateNode } = useDiagramActions();

  // Find the selected node
  const selectedNode = nodes.find((n) => n.selected);

  const handleLabelChange = (newLabel: string) => {
    if (!selectedNode) return;
    updateNode(selectedNode.id, {
      ...selectedNode,
      data: { ...selectedNode.data, label: newLabel },
    });
  };

  const handleClose = () => {
    if (!selectedNode) return;
    updateNode(selectedNode.id, {
      ...selectedNode,
      selected: false,
    });
  };

  return (
    <aside 
      className={`relative rounded-xl border border-slate-800 bg-slate-900/60 shadow-lg backdrop-blur-sm transition-all duration-300 ${
        selectedNode ? 'w-[320px]' : 'w-16'
      }`}
    >
      {/* Vertical "INSPECTOR" text when no node selected */}
      {!selectedNode && (
        <div className="flex items-center justify-center py-8">
          <span className="text-base font-semibold tracking-wider [writing-mode:vertical-lr] rotate-360">
            CONFIG
          </span>
        </div>
      )}

      {/* Content when node is selected */}
      {selectedNode && (
        <div className="p-4">
          {/* Header */}
          <div className="mb-4 flex items-center justify-between">
            <h3 className="text-sm font-semibold text-white">Node Inspector</h3>
            <button
              onClick={handleClose}
              className="rounded-md p-1 text-gray-400 transition-colors hover:bg-slate-800 hover:text-white"
              title="Deselect node"
            >
              <X className="h-4 w-4" />
            </button>
          </div>

          {/* Node Info */}
          <div className="space-y-4">
            {/* Node Type Badge */}
            <div className="flex items-center gap-3 rounded-lg border border-slate-700 bg-slate-800/50 p-3">
              <div className="grid h-10 w-10 shrink-0 place-items-center rounded-md border border-slate-700 bg-slate-900 text-sm font-bold text-gray-300">
                {selectedNode.data.img}
              </div>
              <div>
                <div className="text-xs text-gray-400">Type</div>
                <div className="text-sm font-medium text-white">
                  {selectedNode.type === "awsService" ? "AWS Service" : selectedNode.type}
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
                value={selectedNode.data.label}
                onChange={(e) => handleLabelChange(e.target.value)}
                className="w-full rounded-lg border border-slate-700 bg-slate-800/50 px-3 py-2 text-sm text-white transition-colors focus:border-teal-500 focus:outline-none focus:ring-1 focus:ring-teal-500"
              />
            </div>

            {/* Position Info */}
            <div>
              <label className="mb-2 block text-xs font-semibold uppercase tracking-wider text-gray-500">
                Position
              </label>
              <div className="grid grid-cols-2 gap-2">
                <div className="rounded-lg border border-slate-700 bg-slate-800/50 px-3 py-2">
                  <div className="text-xs text-gray-400">X</div>
                  <div className="text-sm font-medium text-white">
                    {Math.round(selectedNode.position.x)}
                  </div>
                </div>
                <div className="rounded-lg border border-slate-700 bg-slate-800/50 px-3 py-2">
                  <div className="text-xs text-gray-400">Y</div>
                  <div className="text-sm font-medium text-white">
                    {Math.round(selectedNode.position.y)}
                  </div>
                </div>
              </div>
            </div>

            <div>
              <label className="mb-2 block text-xs font-semibold uppercase tracking-wider text-gray-500">
                Node ID
              </label>
              <div className="overflow-hidden rounded-lg border border-slate-700 bg-slate-800/50 px-3 py-2">
                <div className="truncate text-xs text-gray-400">{selectedNode.id}</div>
              </div>
            </div>
          </div>
        </div>
      )}
    </aside>
  );
}