"use client";
import { useDiagramActions, useDiagramState } from "@/lib/stores/diagramStore";
import type { DiagramEdge, DiagramNode, PaletteItem } from "@/lib/types";
import type { Connection, EdgeChange, NodeChange, NodeProps, NodeTypes } from "@xyflow/react";
import { Background, BackgroundVariant, Controls, ReactFlow, useReactFlow } from "@xyflow/react";
import React, { useCallback, useEffect, useMemo } from "react";
import Palette from "./Palette";
import TopNav from "./TopNav";
import AwsServiceNode from "./nodes/AwsServiceNode";

const DND_MIME = "application/x-palette-item";

export default function DiagramEditor({
  projectId,
  diagramId,
}: {
  projectId: string;
  diagramId: string;
}) {
  const { screenToFlowPosition } = useReactFlow();

  // ----- Zustand state -----
  const { nodes, edges, dirty, isSaving, isLoading } = useDiagramState();
  const { addNode, applyNodeChanges, applyEdgeChanges, addEdgeFromConnection, saveDiagram, setContext, loadDiagram } = useDiagramActions();

  console.log(nodes, edges, dirty);

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
    <div className="min-h-screen px-12 py-8">
      <TopNav onSave={onSave} />
      
      {/* Main container - matching project page style */}
      <div>
        <div className="flex gap-4 p-4">
          <Palette />
          
          {/* Canvas area - bigger, clean design */}
          <div className="flex-1 overflow-hidden rounded-xl border border-slate-800 bg-slate-950/60">
            <div className="h-[80vh]">
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
                <Background 
                  variant={BackgroundVariant.Dots} 
                  gap={20} 
                  size={1.5}
                  
                />
                <Controls />
              </ReactFlow>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}