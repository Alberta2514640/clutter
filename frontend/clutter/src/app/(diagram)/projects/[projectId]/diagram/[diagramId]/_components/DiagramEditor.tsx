"use client";

import type { Connection, EdgeChange, IsValidConnection, NodeChange, NodeProps, NodeTypes } from "@xyflow/react";
import { addEdge, applyEdgeChanges, applyNodeChanges, Background, BackgroundVariant, Controls, Panel, ReactFlow, useReactFlow } from "@xyflow/react";
import React, { useCallback, useEffect, useMemo, useState } from "react";

import Palette from "./Palette";
import TopNav from "./TopNav";
import AwsServiceNode from "./nodes/AwsServiceNode";
import ConfigPanel from "./nodes/ConfigPanel";
import LogsPanel, { LogEntry } from "./nodes/LogsPanel";

import { useDiagram, useUpdateDiagramData } from "@/lib/features/diagram/hooks";
import { useSupportedResources } from "@/lib/features/resources/hooks";
import type { DiagramEdge, DiagramNode } from "@/lib/features/diagram/types";
import { useDiagramEditor, useDiagramEditorActions } from "@/lib/features/diagram/uiStore";
import { useMe } from "@/lib/features/user/hooks";
import { useRouter } from "next/navigation";

export type PaletteItem = {
  label: string;
  img: string;
};

const DND_MIME = "application/x-palette-item";

export default function DiagramEditor({ projectId, diagramId }: { projectId: string; diagramId: string }) {
  const router = useRouter();

  const meQ = useMe();
  const token = meQ.data?.token ?? null;

  const { screenToFlowPosition } = useReactFlow();

  const { data: supportedResources } = useSupportedResources();
  const diagramQ = useDiagram(token, projectId, diagramId);
  console.log(diagramQ.data);
  const saveM = useUpdateDiagramData(token);

  const editor = useDiagramEditor(diagramId);
  const { ensure, reset, hydrateFromServer, setNodes, setEdges, setNodesWithoutDirty, setEdgesWithoutDirty, setName, markClean } = useDiagramEditorActions();

  const [logs, setLogs] = useState<LogEntry[]>([]); //temp probably be changed after deploy flow

  useEffect(() => {
    ensure(diagramId);
  }, [diagramId, ensure]);

  useEffect(() => {
    if (!diagramQ.data) return;

    hydrateFromServer(diagramId, diagramQ.data.name ?? "", diagramQ.data.data?.nodes ?? [], diagramQ.data.data?.edges ?? []);
  }, [diagramId, diagramQ.data, hydrateFromServer]);

  const nodes = useMemo(() => editor?.nodes ?? [], [editor?.nodes]);
  const edges = useMemo(() => editor?.edges ?? [], [editor?.edges]);
  const name = editor?.name ?? "";
  const dirty = !!editor?.dirty;

  const isLoading = diagramQ.isLoading;
  const isSaving = saveM.isPending;

  const isValidConnection: IsValidConnection<DiagramEdge> = useCallback(
    (connection) => {
      const source = nodes.find((n) => n.id === connection.source);
      const target = nodes.find((n) => n.id === connection.target);

      if (!source || !target) return false;

      const sourceLabel = source.data.label.toLowerCase();
      const targetLabel = target.data.label.toLowerCase();

      if (targetLabel.includes("api gateway")) return false;
      if (sourceLabel.includes("dynamodb")) return false;
      if (sourceLabel.includes("s3")) return false;
      if (sourceLabel.includes("lambda") && targetLabel.includes("api gateway")) return false;
      if (sourceLabel.includes("ec2") && targetLabel.includes("api gateway")) return false;
      if (connection.source === connection.target) return false;

      return true;
    },
    [nodes],
  );

  const onBack = React.useCallback(() => {
    // Optional: guard if you don't want accidental loss
    if (dirty && !confirm("You have unsaved changes. Leave without saving?")) return;

    reset(diagramId);
    router.back();
  }, [dirty, reset, diagramId, router]);

  // React Flow registry
  const nodeTypes = useMemo<NodeTypes>(
    () => ({
      awsService: AwsServiceNode as React.ComponentType<NodeProps>,
    }),
    [],
  );

  // ---------------------------
  // ReactFlow change handlers
  // ---------------------------
  const onNodesChange = useCallback(
    (changes: NodeChange<DiagramNode>[]) => {
      // Only these change types require saving
      const requiresSave = changes.some((change) => change.type === "position" || change.type === "dimensions" || change.type === "add" || change.type === "remove" || change.type === "replace");

      const next = applyNodeChanges(changes, nodes);

      if (requiresSave) {
        setNodes(diagramId, next);
      } else {
        setNodesWithoutDirty(diagramId, next);
      }
    },
    [diagramId, nodes, setNodes, setNodesWithoutDirty],
  );

  const onEdgesChange = useCallback(
    (changes: EdgeChange<DiagramEdge>[]) => {
      // Only these change types require saving
      const requiresSave = changes.some((change) => change.type !== "select");

      const next = applyEdgeChanges(changes, edges);

      if (requiresSave) {
        setEdges(diagramId, next);
      } else {
        setEdgesWithoutDirty(diagramId, next);
      }
    },
    [diagramId, edges, setEdges, setEdgesWithoutDirty],
  );

  const onConnect = useCallback(
    (params: Connection) => {
      const next = addEdge({ ...params }, edges);
      setEdges(diagramId, next);
    },
    [diagramId, edges, setEdges],
  );

  const onDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
  }, []);

  const onDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      const raw = e.dataTransfer.getData(DND_MIME);
      if (!raw) return;

      const item: PaletteItem = JSON.parse(raw);
      const position = screenToFlowPosition({ x: e.clientX, y: e.clientY });

      // Pre-populate variables with non-null defaults from the resource catalog
      const resourceDef = supportedResources?.find((r) => r.label === item.label);
      const defaultVariables: Record<string, unknown> = {};
      if (resourceDef) {
        for (const v of resourceDef.variables) {
          if (v.default !== null) {
            defaultVariables[v.name] = v.default;
          }
        }
      }

      const newNode: DiagramNode = {
        id: crypto.randomUUID(),
        type: "awsService",
        position,
        data: {
          label: item.label,
          img: item.img,
          ...(Object.keys(defaultVariables).length > 0 && { variables: defaultVariables }),
        },
      };

      setNodes(diagramId, [...nodes, newNode]);
    },
    [diagramId, nodes, screenToFlowPosition, setNodes, supportedResources],
  );

  // ---------------------------
  // Save handler (uses new hook signature)
  // ---------------------------
  const [showSaved, setShowSaved] = useState(false);

  const onSave = useCallback(async () => {
    if (!token) return;

    await saveM.mutateAsync({
      projectId,
      diagramId,
      name: editor?.name?.trim() || "Untitled diagram",
      nodes,
      edges,
    });

    markClean(diagramId);
    setShowSaved(true);
    setTimeout(() => setShowSaved(false), 2000);
  }, [token, saveM, projectId, diagramId, editor?.name, nodes, edges, markClean]);

  return (
    <div className="h-screen w-screen overflow-hidden">
      <div className="flex h-full w-full">
        <Palette />

        <div className="relative flex-1">
          <ReactFlow
            colorMode="dark"
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onDragOver={onDragOver}
            onDrop={onDrop}
            nodeTypes={nodeTypes}
            snapToGrid
            snapGrid={[20, 20]}
            isValidConnection={isValidConnection}>
            <Panel position="top-left" className="w-full pr-5">
              <TopNav diagramName={name} onNameChange={(n) => setName(diagramId, n)} onSave={onSave} onBack={onBack} dirty={dirty} isSaving={isSaving} />
            </Panel>

            <Background variant={BackgroundVariant.Dots} gap={20} size={1.5} />

            <Controls orientation="horizontal" className="[&_button]:!w-10 [&_button]:!h-10 [&_button]:!min-w-10 [&_button]:!min-h-10" showInteractive={false} />
          </ReactFlow>

          {isLoading && (
            <div className="absolute inset-0 grid place-items-center bg-black/40">
              <div className="rounded-lg bg-neutral-900 px-4 py-2 text-sm">Loading diagram…</div>
            </div>
          )}
          {isSaving && (
            <div className="absolute bottom-6 left-1/2 -translate-x-1/2 z-50 rounded-lg bg-slate-900 border border-slate-700 px-4 py-2 text-sm text-white shadow-lg flex items-center gap-2">
              <div className="h-3 w-3 animate-spin rounded-full border-2 border-white border-t-transparent" />
              Saving…
            </div>
          )}
          {showSaved && !isSaving && (
            <div className="absolute bottom-6 left-1/2 -translate-x-1/2 z-50 rounded-lg bg-emerald-900/80 border border-emerald-700 px-4 py-2 text-sm text-emerald-100 shadow-lg flex items-center gap-2">
              <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
              </svg>
              Saved
            </div>
          )}
          <LogsPanel logs={logs} />
        </div>

        <ConfigPanel diagramId={diagramId} />
      </div>
    </div>
  );
}
