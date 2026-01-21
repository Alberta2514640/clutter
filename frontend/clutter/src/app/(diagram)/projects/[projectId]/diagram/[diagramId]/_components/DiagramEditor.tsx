"use client";
import { useDiagramActions, useDiagramState } from "@/lib/stores/diagramStore";
import type { DiagramEdge, DiagramNode, PaletteItem } from "@/lib/types";
import type { Connection, EdgeChange, NodeChange, NodeProps, NodeTypes } from "@xyflow/react";
import { Background, BackgroundVariant, Controls, ReactFlow, useReactFlow } from "@xyflow/react";
import React, { useCallback, useEffect, useMemo } from "react";
import Palette from "./Palette";
import TopNav from "./TopNav";
import AwsServiceNode from "./nodes/AwsServiceNode";
import ConfigPanel from "./nodes/ConfigPanel";

const DND_MIME = "application/x-palette-item";

export default function DiagramEditor({ projectId, diagramId }: { projectId: string; diagramId: string }) {
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
<<<<<<< HEAD
    <div className=" h-screen w-screen overflow-hidden">

      {/* Fullscreen React Flow Canvas */}
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
              <TopNav onSave={onSave} />
            </Panel>
            <Background 
              variant={BackgroundVariant.Dots} 
              gap={20} 
              size={1.5}
            />
            <Controls 
              orientation="horizontal"
              className="[&_button]:!w-10 [&_button]:!h-10 [&_button]:!min-w-10 [&_button]:!min-h-10"
              showInteractive={false}
            />
          </ReactFlow>
        </div>
        <ConfigPanel />
=======
    <div className="relative h-screen w-screen overflow-hidden">
      {/* Top Nav - Fixed at top */}
      <div className="absolute top-0 left-0 right-0 z-30">
        <TopNav onSave={onSave} />
      </div>

      {/* Floating Palette - Fixed position on left */}
      <div className="absolute top-20 left-4 z-20">
        <Palette />
      </div>

      <div className="absolute top-20 right-4 z-20">
        <ConfigPanel />
      </div>

      {/* Fullscreen React Flow Canvas */}
      <div className="h-full w-full">
        <ReactFlow colorMode="dark" nodes={nodes} edges={edges} onNodesChange={onNodesChange} onEdgesChange={onEdgesChange} onConnect={onConnect} onDragOver={onDragOver} onDrop={onDrop} nodeTypes={nodeTypes} snapToGrid snapGrid={[20, 20]}>
          <Background variant={BackgroundVariant.Dots} gap={20} size={1.5} />
          <Controls orientation="horizontal" className="[&_button]:!w-10 [&_button]:!h-10 [&_button]:!min-w-10 [&_button]:!min-h-10" showInteractive={false} />
        </ReactFlow>
>>>>>>> origin/main
      </div>
    </div>
  );
}
