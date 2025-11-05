"use client";

import React, { useCallback, useEffect, useMemo } from "react";
import ReactFlow, { addEdge, Background, Controls, MiniMap, useEdgesState, useNodesState, Connection, Edge as RFEdge, Node as RFNode } from "reactflow";
import "reactflow/dist/style.css";

import { useEditor } from "@/lib/state/editorStore";
import { isAllowed } from "@/lib/graph/rules";
import { wouldCreateCycle } from "@/lib/graph/cycles";
import type { Node as DomainNode, Edge as DomainEdge } from "@/lib/types";
import { logInfo, logWarn, logError, logSuccess, logDebug } from "@/components/diagram/Console";

// ---- helpers: map domain → React Flow ----
function toRFNodes(nodes: DomainNode[]): RFNode[] {
  return nodes.map((n) => ({
    id: n.nodeId,
    type: "default",
    position: (n.ui as any)?.position ?? { x: Math.random() * 200, y: Math.random() * 120 },
    data: { resourceType: n.resourceType, spec: n.spec, label: n.resourceType },
  }));
}

function toRFEdges(edges: DomainEdge[]): RFEdge[] {
  return edges.map((e) => ({
    id: e.edgeId,
    source: e.fromNodeId,
    target: e.toNodeId,
    label: e.relation,
  }));
}

export default function Canvas() {
  const { bundle, upsertEdge, upsertNode, removeById } = useEditor();

  const initialRFNodes = useMemo<RFNode[]>(() => toRFNodes(bundle?.nodes ?? []), [bundle?.nodes]);
  const initialRFEdges = useMemo<RFEdge[]>(() => toRFEdges(bundle?.edges ?? []), [bundle?.edges]);

  const [nodes, setNodes, onNodesChange] = useNodesState(initialRFNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialRFEdges);

  // Log when the bundle/canvas loads or changes
  useEffect(() => {
    if (!bundle) return;
    logInfo("Canvas loaded", {
      canvasId: bundle.canvas.canvasId,
      name: bundle.canvas.name,
      nodes: bundle.nodes.length,
      edges: bundle.edges.length,
    });
  }, [bundle?.canvas.canvasId, bundle?.canvas.name, bundle?.nodes?.length, bundle?.edges?.length]);

  // keep domain store in sync when positions change + log moves
  const handleNodesChange = useCallback(
    (changes: Parameters<typeof onNodesChange>[0]) => {
      onNodesChange(changes);
      const movedIds = new Set(changes.filter((c) => c.type === "position" || (c.type === "dimensions" && (c as any).dragging)).map((c: any) => c.id));
      if (!movedIds.size) return;

      const byId = new Map(nodes.map((n) => [n.id, n]));
      movedIds.forEach((id) => {
        const n = byId.get(id);
        if (!n) return;
        upsertNode({
          nodeId: n.id,
          canvasId: bundle?.canvas.canvasId ?? "",
          resourceType: (n.data as any)?.resourceType ?? "Unknown",
          spec: (n.data as any)?.spec ?? {},
          ui: { ...(n as any).ui, position: n.position },
        } as any);
        logDebug("Node moved", { nodeId: n.id, position: n.position });
      });
    },
    [onNodesChange, nodes, upsertNode, bundle?.canvas.canvasId]
  );

  const onConnect = useCallback(
    (connection: Connection) => {
      const from = nodes.find((n) => n.id === connection.source);
      const to = nodes.find((n) => n.id === connection.target);
      if (!from || !to) return;

      const relation = (connection as any).label || "invokes";
      const valid = isAllowed((from.data as any)?.resourceType, relation, (to.data as any)?.resourceType);

      const createsCycle = wouldCreateCycle(
        edges.map((e) => ({ from: e.source, to: e.target })),
        { from: connection.source!, to: connection.target! }
      );

      if (!valid) {
        logWarn("Connection rejected: rule violation", {
          from: from.data.resourceType,
          to: to.data.resourceType,
          relation,
        });
        return;
      }
      if (createsCycle) {
        logWarn("Connection rejected: would create cycle", {
          from: from.id,
          to: to.id,
        });
        return;
      }

      const newEdge: RFEdge = {
        id: `e-${connection.source}-${connection.target}-${Date.now()}`,
        source: connection.source!,
        target: connection.target!,
        label: relation,
      };
      setEdges((eds) => addEdge(newEdge, eds));
      upsertEdge({
        edgeId: newEdge.id,
        fromNodeId: newEdge.source,
        toNodeId: newEdge.target,
        relation,
      } as any);

      logSuccess("Edge added", {
        id: newEdge.id,
        from: from.data.resourceType,
        to: to.data.resourceType,
        relation,
      });
    },
    [nodes, edges, setEdges, upsertEdge]
  );

  const onEdgeDelete = useCallback(
    (edgeIds: string[]) => {
      edgeIds.forEach((id) => {
        removeById(id);
        logError("Edge deleted", { edgeId: id });
      });
      setEdges((eds) => eds.filter((e) => !edgeIds.includes(e.id)));
    },
    [setEdges, removeById]
  );

  return (
    <div className="h-[calc(100vh-4rem)] w-full">
      <ReactFlow nodes={nodes} edges={edges} onNodesChange={handleNodesChange} onEdgesChange={onEdgesChange} onConnect={onConnect} onEdgesDelete={(eds) => onEdgeDelete(eds.map((e) => e.id))} fitView>
        <MiniMap />
        <Controls />
        <Background variant="dots" gap={12} size={1} />
      </ReactFlow>
    </div>
  );
}
