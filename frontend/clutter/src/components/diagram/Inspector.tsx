"use client";

import React, { useEffect, useMemo, useState } from "react";
import { useEditor } from "@/lib/state/editorStore";
import type { Node as DomainNode } from "@/lib/types";
import { logSuccess, logError, logInfo } from "@/components/diagram/Console";

type Editable = {
  nodeId: string;
  resourceType: string;
  specText: string; // JSON text for editing
  moduleAlias?: string;
  moduleSource?: string;
  moduleVersion?: string;
  uiLabel?: string;
};

export default function Inspector() {
  const { bundle, upsertNode, removeById } = useEditor();

  const nodes = bundle?.nodes ?? [];
  const [selectedId, setSelectedId] = useState<string | undefined>(nodes[0]?.nodeId);
  const selectedNode = useMemo(() => nodes.find((n) => n.nodeId === selectedId), [nodes, selectedId]);

  const [model, setModel] = useState<Editable | null>(null);

  // initialize/refresh form when selection or source node changes
  useEffect(() => {
    if (!selectedNode) {
      setModel(null);
      return;
    }
    setModel(toEditable(selectedNode));
  }, [selectedNode?.nodeId, selectedNode?.resourceType, selectedNode?.spec, selectedNode?.iac, selectedNode?.ui]);

  // if nodes list changes (add/remove), keep selection sane
  useEffect(() => {
    if (!selectedId && nodes.length) setSelectedId(nodes[0].nodeId);
    if (selectedId && !nodes.some((n) => n.nodeId === selectedId)) {
      setSelectedId(nodes[0]?.nodeId);
    }
  }, [nodes, selectedId]);

  if (!bundle) return null;

  return (
    <div className="w-full md:w-96 border rounded-xl bg-neutral-950/90 text-neutral-100">
      <div className="px-4 py-3 border-b border-neutral-800 flex items-center gap-2">
        <span className="text-sm font-semibold text-neutral-300">Inspector</span>
        <div className="ml-auto text-xs text-neutral-400">
          {bundle.canvas.name} · {nodes.length} nodes
        </div>
      </div>

      <div className="p-4 space-y-4">
        {/* Node selector */}
        <div className="space-y-1">
          <label className="text-sm text-neutral-300">Node</label>
          <select className="w-full bg-neutral-900 border border-neutral-700 rounded px-2 py-1.5 text-sm" value={selectedId ?? ""} onChange={(e) => setSelectedId(e.target.value || undefined)}>
            {nodes.map((n) => (
              <option key={n.nodeId} value={n.nodeId}>
                {n.nodeId} — {n.resourceType}
              </option>
            ))}
            {nodes.length === 0 && <option value="">(no nodes)</option>}
          </select>
        </div>

        {!model ? (
          <p className="text-sm text-neutral-400">Select a node to edit.</p>
        ) : (
          <>
            {/* Resource type */}
            <div className="space-y-1">
              <label className="text-sm text-neutral-300">Resource Type</label>
              <input
                className="w-full bg-neutral-900 border border-neutral-700 rounded px-2 py-1.5 text-sm"
                value={model.resourceType}
                onChange={(e) => setModel({ ...model, resourceType: e.target.value })}
                placeholder="Lambda, DynamoDB, S3, APIGateway…"
              />
            </div>

            {/* UI label */}
            <div className="space-y-1">
              <label className="text-sm text-neutral-300">UI Label (optional)</label>
              <input className="w-full bg-neutral-900 border border-neutral-700 rounded px-2 py-1.5 text-sm" value={model.uiLabel ?? ""} onChange={(e) => setModel({ ...model, uiLabel: e.target.value })} placeholder="Shown on the node" />
              {selectedNode?.ui?.position ? (
                <p className="text-[11px] text-neutral-500">
                  Position: x={selectedNode.ui.position.x}, y={selectedNode.ui.position.y}
                </p>
              ) : (
                <p className="text-[11px] text-neutral-500">Position: (not set)</p>
              )}
            </div>

            {/* IaC fields */}
            <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
              <div className="space-y-1">
                <label className="text-sm text-neutral-300">Module Alias</label>
                <input className="w-full bg-neutral-900 border border-neutral-700 rounded px-2 py-1.5 text-sm" value={model.moduleAlias ?? ""} onChange={(e) => setModel({ ...model, moduleAlias: e.target.value })} placeholder="aws_lambda_fn" />
              </div>
              <div className="space-y-1">
                <label className="text-sm text-neutral-300">Module Version</label>
                <input className="w-full bg-neutral-900 border border-neutral-700 rounded px-2 py-1.5 text-sm" value={model.moduleVersion ?? ""} onChange={(e) => setModel({ ...model, moduleVersion: e.target.value })} placeholder="~> 1.2.3" />
              </div>
              <div className="md:col-span-2 space-y-1">
                <label className="text-sm text-neutral-300">Module Source</label>
                <input
                  className="w-full bg-neutral-900 border border-neutral-700 rounded px-2 py-1.5 text-sm"
                  value={model.moduleSource ?? ""}
                  onChange={(e) => setModel({ ...model, moduleSource: e.target.value })}
                  placeholder="git::https://github.com/acme/terraform-aws-lambda"
                />
              </div>
            </div>

            {/* Spec JSON */}
            <div className="space-y-1">
              <div className="flex items-center gap-2">
                <label className="text-sm text-neutral-300">Spec (JSON)</label>
                <button
                  className="ml-auto text-xs px-2 py-0.5 rounded border border-neutral-600 hover:border-neutral-400"
                  onClick={() => {
                    try {
                      const pretty = JSON.stringify(JSON.parse(model.specText || "{}"), null, 2);
                      setModel({ ...model, specText: pretty });
                    } catch {
                      // ignore
                    }
                  }}>
                  Format
                </button>
              </div>
              <textarea
                className="w-full h-40 bg-neutral-900 border border-neutral-700 rounded px-2 py-1.5 text-sm font-mono"
                value={model.specText}
                onChange={(e) => setModel({ ...model, specText: e.target.value })}
                placeholder='{"runtime":"nodejs20.x","memory":256}'
              />
              <JSONHint text={model.specText} />
            </div>

            {/* Actions */}
            <div className="flex items-center gap-2 pt-2">
              <button
                className="px-3 py-1.5 text-sm rounded border border-emerald-600 text-emerald-300 hover:border-emerald-400"
                onClick={() => {
                  try {
                    const parsed = parseJSON(model.specText);
                    const next: DomainNode = {
                      nodeId: model.nodeId,
                      canvasId: bundle.canvas.canvasId,
                      resourceType: model.resourceType || selectedNode!.resourceType,
                      spec: parsed,
                      iac:
                        model.moduleAlias || model.moduleSource || model.moduleVersion
                          ? {
                              moduleAlias: model.moduleAlias || "",
                              moduleSource: model.moduleSource || undefined,
                              version: model.moduleVersion || undefined,
                            }
                          : selectedNode!.iac,
                      ui: {
                        ...(selectedNode?.ui || {}),
                        label: model.uiLabel || undefined,
                        position: selectedNode?.ui?.position, // position managed by Canvas
                      },
                      etag: selectedNode?.etag,
                      version: selectedNode?.version,
                    };
                    upsertNode(next);
                    logSuccess("Node saved", { nodeId: model.nodeId });
                  } catch (e: any) {
                    logError("Invalid JSON in spec", { error: String(e?.message || e) });
                    alert("Spec must be valid JSON.");
                  }
                }}>
                Save
              </button>

              <button
                className="px-3 py-1.5 text-sm rounded border border-neutral-600 hover:border-neutral-400"
                onClick={() => {
                  if (!selectedNode) return;
                  setModel(toEditable(selectedNode));
                  logInfo("Reverted changes", { nodeId: selectedNode.nodeId });
                }}>
                Reset
              </button>

              <button
                className="ml-auto px-3 py-1.5 text-sm rounded border border-red-600 text-red-300 hover:border-red-400"
                onClick={() => {
                  if (!selectedNode) return;
                  const ok = confirm(`Delete node ${selectedNode.nodeId}? This will also remove connected edges.`);
                  if (ok) {
                    removeById(selectedNode.nodeId);
                    logError("Node deleted", { nodeId: selectedNode.nodeId });
                  }
                }}>
                Delete
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

/* ---------------- helpers ---------------- */

function toEditable(n: DomainNode): Editable {
  return {
    nodeId: n.nodeId,
    resourceType: n.resourceType,
    specText: safeStringify(n.spec, 2),
    moduleAlias: n.iac?.moduleAlias,
    moduleSource: n.iac?.moduleSource,
    moduleVersion: n.iac?.version,
    uiLabel: (n.ui as any)?.label,
  };
}

function parseJSON(text: string) {
  if (!text.trim()) return {};
  return JSON.parse(text);
}

function safeStringify(v: unknown, indent = 2) {
  try {
    return JSON.stringify(v ?? {}, null, indent);
  } catch {
    return "{}";
  }
}

function JSONHint({ text }: { text: string }) {
  try {
    if (text.trim()) JSON.parse(text);
    return <p className="text-[11px] text-emerald-400">JSON OK</p>;
  } catch (e: any) {
    return <p className="text-[11px] text-red-400">JSON error: {String(e?.message || e)}</p>;
  }
}
