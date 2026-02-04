"use client";

import type { Connection, EdgeChange, NodeChange, NodeProps, NodeTypes } from "@xyflow/react";
import {
  addEdge,
  applyEdgeChanges,
  applyNodeChanges,
  Background,
  BackgroundVariant,
  Controls,
  Panel,
  ReactFlow,
  useReactFlow,
} from "@xyflow/react";
import React, { useCallback, useEffect, useMemo } from "react";

import Palette from "./Palette";
import TopNav from "./TopNav";
import AwsServiceNode from "./nodes/AwsServiceNode";
import ConfigPanel from "./nodes/ConfigPanel";

import { useDiagram, useUpdateDiagramData } from "@/lib/features/diagram/hooks";
import type { DiagramEdge, DiagramNode } from "@/lib/features/diagram/types";
import { useDiagramEditor, useDiagramEditorActions } from "@/lib/features/diagram/uiStore";
import { useMe } from "@/lib/features/user/hooks";

export type PaletteItem = {
  label: string;
  img: string;
};

const DND_MIME = "application/x-palette-item";

export default function DiagramEditor({ projectId, diagramId }: { projectId: string; diagramId: string }) {
  const meQ = useMe();
  const token = meQ.data?.token ?? null;

  const { screenToFlowPosition } = useReactFlow();

  // ---------------------------
  // Server state (React Query)
  // ---------------------------
  const diagramQ = useDiagram(token, projectId, diagramId);
  const saveM = useUpdateDiagramData(token);

  // ---------------------------
  // Editor draft state (Zustand)
  // ---------------------------
  const editor = useDiagramEditor(diagramId);
  const { ensure, hydrateFromServer, setNodes, setEdges, setNodesWithoutDirty, setEdgesWithoutDirty, markClean } = useDiagramEditorActions();

  console.log(editor)

  useEffect(() => {
    ensure(diagramId);
  }, [diagramId, ensure]);

  //IMPORTANT: hydrate from normalized uiLayout (NOT diagramQ.data.nodes)
  useEffect(() => {
    if (!diagramQ.data) return;

    hydrateFromServer(
      diagramId,
      diagramQ.data.uiLayout?.nodes ?? [],
      diagramQ.data.uiLayout?.edges ?? []
    );
  }, [diagramId, diagramQ.data, hydrateFromServer]);

  const nodes = useMemo(() => editor?.nodes ?? [], [editor?.nodes]);
  const edges = useMemo(() => editor?.edges ?? [], [editor?.edges]);
  const dirty = !!editor?.dirty;

  const isLoading = diagramQ.isLoading;
  const isSaving = saveM.isPending;

  // React Flow registry
  const nodeTypes = useMemo<NodeTypes>(
    () => ({
      awsService: AwsServiceNode as React.ComponentType<NodeProps>,
    }),
    []
  );

  // ---------------------------
  // ReactFlow change handlers
  // ---------------------------
  const onNodesChange = useCallback(
    (changes: NodeChange<DiagramNode>[]) => {
      // Only these change types require saving
      const requiresSave = changes.some(
        (change) => 
          change.type === 'position' ||
          change.type === 'dimensions' ||
          change.type === 'add' ||
          change.type === 'remove' ||
          change.type === 'replace'
      );
      
      const next = applyNodeChanges(changes, nodes);
      
      if (requiresSave) {
        setNodes(diagramId, next);
      } else {
        setNodesWithoutDirty(diagramId, next);
      }
    },
    [diagramId, nodes, setNodes, setNodesWithoutDirty]
  );

  const onEdgesChange = useCallback(
    (changes: EdgeChange<DiagramEdge>[]) => {
      // Only these change types require saving
      const requiresSave = changes.some(
        (change) => 
          change.type !== 'select'
      );
      
      const next = applyEdgeChanges(changes, edges);
      
      if (requiresSave) {
        setEdges(diagramId, next);
      } else {
        setEdgesWithoutDirty(diagramId, next);
      }
    },
    [diagramId, edges, setEdges, setEdgesWithoutDirty]
  );

  const onConnect = useCallback(
    (params: Connection) => {
      const next = addEdge({ ...params }, edges);
      setEdges(diagramId, next);
    },
    [diagramId, edges, setEdges]
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

      const newNode: DiagramNode = {
        id: crypto.randomUUID(),
        type: "awsService",
        position,
        data: { label: item.label, img: item.img },
      };

      setNodes(diagramId, [...nodes, newNode]);
    },
    [diagramId, nodes, screenToFlowPosition, setNodes]
  );

  // ---------------------------
  // Save handler (uses new hook signature)
  // ---------------------------
  const onSave = useCallback(async () => {
    if (!token) return;

    // backend requires name; take it from the diagram detail
    const name = diagramQ.data?.name ?? "UpdatedWithNodes";

    await saveM.mutateAsync({
      projectId,
      diagramId,
      name,
      nodes,
      edges,
    });

    markClean(diagramId);
  }, [token, saveM, projectId, diagramId, diagramQ.data?.name, nodes, edges, markClean]);

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
          >
            <Panel position="top-right">
              <TopNav onSave={onSave} dirty={dirty} isSaving={isSaving} />
            </Panel>

            <Background variant={BackgroundVariant.Dots} gap={20} size={1.5} />

            <Controls
              orientation="horizontal"
              className="[&_button]:!w-10 [&_button]:!h-10 [&_button]:!min-w-10 [&_button]:!min-h-10"
              showInteractive={false}
            />
          </ReactFlow>

          {isLoading && (
            <div className="absolute inset-0 grid place-items-center bg-black/40">
              <div className="rounded-lg bg-neutral-900 px-4 py-2 text-sm">Loading diagram…</div>
            </div>
          )}
        </div>

        <ConfigPanel diagramId={diagramId} />
      </div>
    </div>
  );
}