"use client";

import { Play, Settings, Trash2, Upload, X } from "lucide-react";
import { useCallback, useMemo } from "react";
import Image from "next/image";

import type { DiagramNode } from "@/lib/features/diagram/types";
import { useDiagramEditor, useDiagramEditorActions } from "@/lib/features/diagram/uiStore";
import { useSupportedResources } from "@/lib/features/resources/hooks";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

export default function ConfigPanel({ diagramId }: { diagramId: string }) {
  const editor = useDiagramEditor(diagramId);
  const { setNodes, setEdges } = useDiagramEditorActions();
  const { data: supportedResources } = useSupportedResources();

  const nodes = useMemo(() => editor?.nodes ?? [], [editor?.nodes]);
  const edges = useMemo(() => editor?.edges ?? [], [editor?.edges]);

  const selectedNode = useMemo(() => nodes.find((n) => n.selected), [nodes]);
  const showContent = !!selectedNode;
  const isEc2Node = selectedNode?.data.img?.includes("ec2") ?? false;

  const resourceDef = useMemo(
    () => supportedResources?.find((r) => r.label === selectedNode?.data.label),
    [supportedResources, selectedNode?.data.label],
  );

  const patchSelectedNode = useCallback(
    (patch: (node: DiagramNode) => DiagramNode) => {
      if (!selectedNode) return;
      const next = nodes.map((n) => (n.id === selectedNode.id ? patch(n) : n));
      setNodes(diagramId, next);
    },
    [diagramId, nodes, selectedNode, setNodes],
  );

  const handleClose = () => {
    patchSelectedNode((n) => ({ ...n, selected: false }));
  };

  const handleVariableChange = (varName: string, value: unknown) => {
    patchSelectedNode((n) => ({
      ...n,
      data: {
        ...n.data,
        variables: { ...(n.data.variables ?? {}), [varName]: value },
      },
    }));
  };

  const handleAnsiblePlaybookUpload = (file: File | null) => {
    if (!file) return;
    patchSelectedNode((n) => ({ ...n, data: { ...n.data, ansiblePlaybookName: file.name } }));
  };

  const handleRunPlaybook = () => {
    if (!selectedNode?.data.ansiblePlaybookName) return;
    window.alert(`Ansible playbook "${selectedNode.data.ansiblePlaybookName}" is ready to run on this EC2 block.`);
  };

  const handleDeleteSelected = useCallback(() => {
    if (!selectedNode) return;
    const nodeId = selectedNode.id;
    setNodes(diagramId, nodes.filter((n) => n.id !== nodeId));
    setEdges(diagramId, edges.filter((e) => e.source !== nodeId && e.target !== nodeId));
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
          <div className="h-[calc(100%-3.5rem)] flex flex-col">
            <div className="flex-1 overflow-y-auto p-4 space-y-4">
              {/* Node Type Badge */}
              <div className="flex items-center gap-3 rounded-lg border border-slate-800 bg-slate-900/40 p-3">
                <div className="grid h-10 w-10 shrink-0 place-items-center rounded-md border border-slate-800 bg-slate-950">
                  <Image src={selectedNode!.data.img as string} alt="" width={24} height={24} unoptimized />
                </div>
                <div className="min-w-0">
                  <div className="text-xs text-gray-400">Type</div>
                  <div className="truncate text-sm font-medium text-white">{selectedNode!.data.label}</div>
                </div>
              </div>

              {/* Dynamic Variable Fields */}
              {resourceDef && resourceDef.variables.length > 0 && (
                <div>
                  <label className="mb-2 block text-xs font-semibold uppercase tracking-wider text-gray-500">Configuration</label>
                  <div className="space-y-3">
                    {resourceDef.variables.map((v) => {
                      const val = selectedNode!.data.variables?.[v.name];
                      const placeholder = v.default != null ? String(v.default) : "";
                      return (
                        <div key={v.name}>
                          <Label className="mb-1 flex items-center gap-1 text-xs text-slate-400">
                            {v.name}
                            {v.required && <span className="text-red-400">*</span>}
                          </Label>
                          {v.type === "boolean" ? (
                            <Select
                              value={val === undefined ? "" : String(val)}
                              onValueChange={(s) => handleVariableChange(v.name, s === "true")}
                            >
                              <SelectTrigger className="h-8 border-slate-800 bg-slate-900/40 text-sm text-white">
                                <SelectValue placeholder="— default —" />
                              </SelectTrigger>
                              <SelectContent>
                                <SelectItem value="true">true</SelectItem>
                                <SelectItem value="false">false</SelectItem>
                              </SelectContent>
                            </Select>
                          ) : (
                            <Input
                              type={v.type === "number" ? "number" : "text"}
                              className="h-8 border-slate-800 bg-slate-900/40 text-sm text-white"
                              placeholder={placeholder || v.description}
                              value={val === undefined ? "" : String(val)}
                              onChange={(e) => {
                                const raw = e.target.value;
                                handleVariableChange(v.name, raw === "" ? undefined : v.type === "number" ? Number(raw) : raw);
                              }}
                            />
                          )}
                          {v.description && (
                            <p className="mt-1 text-[11px] text-slate-500">{v.description}</p>
                          )}
                        </div>
                      );
                    })}
                  </div>
                </div>
              )}

              {/* Ansible — EC2 only */}
              {isEc2Node && (
                <div>
                  <div className="mb-2 block text-xs font-semibold uppercase tracking-wider text-gray-500">Ansible Playbook</div>
                  <div className="space-y-3 rounded-lg border border-slate-800 bg-slate-900/40 p-3">
                    <label
                      htmlFor="ansible-playbook-upload"
                      className="flex cursor-pointer items-center justify-center gap-2 rounded-lg border border-dashed border-slate-700 bg-slate-950/60 px-3 py-3 text-sm font-medium text-slate-200 transition hover:border-teal-500/60 hover:bg-slate-900"
                    >
                      <Upload className="h-4 w-4" />
                      {selectedNode!.data.ansiblePlaybookName ? "Replace Ansible playbook" : "Upload Ansible playbook"}
                    </label>
                    <input id="ansible-playbook-upload" type="file" accept=".yml,.yaml" className="hidden"
                      onChange={(e) => {
                        handleAnsiblePlaybookUpload(e.target.files?.[0] ?? null);
                        e.currentTarget.value = "";
                      }}
                    />
                    <div className="rounded-lg border border-slate-800 bg-slate-950/70 px-3 py-2">
                      <div className="text-xs text-gray-400">Uploaded file</div>
                      <div className="mt-1 text-sm font-medium text-white">
                        {selectedNode!.data.ansiblePlaybookName ?? "No playbook uploaded yet"}
                      </div>
                    </div>
                    <button
                      type="button"
                      onClick={handleRunPlaybook}
                      disabled={!selectedNode!.data.ansiblePlaybookName}
                      className={[
                        "flex w-full items-center justify-center gap-2 rounded-lg border px-3 py-2 text-sm font-medium transition",
                        selectedNode!.data.ansiblePlaybookName
                          ? "border-teal-500/40 bg-teal-500/10 text-teal-100 hover:bg-teal-500/20"
                          : "border-slate-800 bg-slate-950/80 text-slate-500",
                      ].join(" ")}
                    >
                      <Play className="h-4 w-4" />
                      {selectedNode!.data.ansiblePlaybookName ? "Run uploaded playbook" : "Upload a playbook to run"}
                    </button>
                  </div>
                </div>
              )}

            </div>

            {/* Danger Zone — pinned to bottom */}
            <div className="border-t border-slate-800 p-4">
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
                title="Delete selected node"
              >
                <Trash2 className="h-4 w-4" />
                Delete node
              </button>
              <p className="mt-2 text-xs text-slate-400">This will also remove any connections to this node.</p>
            </div>
          </div>
        </>
      )}
    </aside>
  );
}
