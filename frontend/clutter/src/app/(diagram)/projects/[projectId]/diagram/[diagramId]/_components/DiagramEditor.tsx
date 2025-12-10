"use client";

import {
    addEdge, Background, BackgroundVariant, Controls, MiniMap, ReactFlow, useEdgesState, useNodesState, useReactFlow, type Connection,
    type Edge, type Node, type NodeProps, type NodeTypes,
} from "@xyflow/react";
import React, { useCallback, useMemo } from "react";

import Palette from "./Palette";
import AwsServiceNode from "./nodes/AwsServiceNode";
import type { NodeData, PaletteItem } from "./types";

const DND_MIME = "application/x-palette-item";

//projectID and diagramId are the id from the params of the url given by the page
export default function DiagramEditor({ projectId, diagramId, }: { projectId: string; diagramId: string; }) {
  const { screenToFlowPosition } = useReactFlow();

  const [nodes, setNodes, onNodesChange] = useNodesState<Node<NodeData>>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);

  const nodeTypes = useMemo<NodeTypes>(
    () => ({
      awsService: AwsServiceNode as React.ComponentType<NodeProps>,
    }),
    []
  );

  const onConnect = useCallback(
    (params: Connection) => {
      setEdges((eds) =>
        addEdge(
          {
            ...params,
            type: "default",
            style: { stroke: "rgba(100,180,255,0.6)", strokeWidth: 2 },
          },
          eds
        )
      );
    },
    [setEdges]
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

      const newNode: Node<NodeData> = {
        id: crypto.randomUUID(),
        type: "awsService",
        position,
        data: { label: item.label, badge: item.badge, category: item.category },
      };

      setNodes((prev) => [...prev, newNode]);
    },
    [screenToFlowPosition, setNodes]
  );

  const onSave = useCallback(() => {
    console.log("SAVE", { projectId, diagramId, nodes, edges });
    alert("Diagram saved! Check console for details.");
  }, [projectId, diagramId, nodes, edges]);

  return (
    <div className="min-h-screen bg-[linear-gradient(180deg,rgb(10,15,25),rgb(15,20,30))] p-5">
      {/* top bar */}
      <div className="mb-4 flex items-center justify-between">
        <button
          onClick={() => window.history.back()}
          className="rounded-lg border border-white/10 bg-white/5 px-4 py-2 text-sm font-semibold text-white shadow-lg backdrop-blur-sm transition-colors hover:bg-white/10"
        >
          ← Back
        </button>

        <button
          onClick={onSave}
          className="rounded-lg border border-blue-400/20 bg-blue-500/20 px-5 py-2 text-sm font-semibold text-white shadow-lg backdrop-blur-sm transition-colors hover:bg-blue-500/30"
        >
          Save
        </button>
      </div>

      {/* main card */}
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
                <MiniMap pannable zoomable />
              </ReactFlow>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
