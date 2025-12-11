"use client";

import type { Connection, EdgeChange, NodeChange, NodeProps, NodeTypes, } from "@xyflow/react";
import { Background, BackgroundVariant, Controls, ReactFlow, useReactFlow, } from "@xyflow/react";
import React, { useCallback, useEffect, useMemo } from "react";

import { useDiagramStore } from "@/lib/stores/diagramStore";
import type { DiagramEdge, DiagramNode, PaletteItem } from "@/lib/types";


import Palette from "./Palette";
import TopNav from "./TopNav";
import AwsServiceNode from "./nodes/AwsServiceNode";

const DND_MIME = "application/x-palette-item";

export default function DiagramEditor({ projectId, diagramId, }: { projectId: string; diagramId: string;}) 
{
  const { screenToFlowPosition } = useReactFlow();

  // ----- Zustand state -----
  const nodes = useDiagramStore((s) => s.nodes);
  const edges = useDiagramStore((s) => s.edges);

  const setContext = useDiagramStore((s) => s.setContext);
  const applyNodeChanges = useDiagramStore((s) => s.applyNodeChanges);
  const applyEdgeChanges = useDiagramStore((s) => s.applyEdgeChanges);
  const addEdgeFromConnection = useDiagramStore((s) => s.addEdgeFromConnection);
  const addNode = useDiagramStore((s) => s.addNode);

  const loadDiagram = useDiagramStore((s) => s.loadDiagram);
  const saveDiagram = useDiagramStore((s) => s.saveDiagram);
  const isLoading = useDiagramStore((s) => s.isLoading);
  const isSaving = useDiagramStore((s) => s.isSaving);
  const dirty = useDiagramStore((s) => s.dirty);
  const error = useDiagramStore((s) => s.error);

  useEffect(() => {
    setContext(projectId, diagramId);
    loadDiagram(projectId, diagramId);
  }, [projectId, diagramId, setContext, loadDiagram]);

  // ----- React Flow registry -----
  const nodeTypes = useMemo<NodeTypes>(
    () => ({
      awsService: AwsServiceNode as React.ComponentType<NodeProps>,
    }),
    []
  );

  // ----- Handlers -----
  const onNodesChange = useCallback(
    (changes: NodeChange<DiagramNode>[]) => {
      applyNodeChanges(changes);
    },
    [applyNodeChanges]
  );

  const onEdgesChange = useCallback(
    (changes: EdgeChange<DiagramEdge>[]) => {
      applyEdgeChanges(changes);
    },
    [applyEdgeChanges]
  );

  const onConnect = useCallback(
    (params: Connection) => {
      addEdgeFromConnection(params);
    },
    [addEdgeFromConnection]
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

      addNode(newNode);
    },
    [screenToFlowPosition, addNode]
  );

  const onSave = useCallback(() => {
    saveDiagram();
  }, [saveDiagram]);


  return (
    <div className="min-h-screen bg-[linear-gradient(180deg,rgb(10,15,25),rgb(15,20,30))] p-5">
      <TopNav onSave={onSave} />

      <div className="rounded-xl border border-white/10 bg-[rgba(20,25,35,0.4)] p-4 shadow-xl backdrop-blur-sm">
        <div className="flex gap-4">
          <Palette />

          <div className="flex-1 overflow-hidden rounded-xl border border-white/10 bg-[rgba(15,20,30,0.6)]">
            <div className="h-[72vh]">
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
                <Background variant={BackgroundVariant.Dots} gap={20} size={1.5} />
                <Controls />
              </ReactFlow>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}