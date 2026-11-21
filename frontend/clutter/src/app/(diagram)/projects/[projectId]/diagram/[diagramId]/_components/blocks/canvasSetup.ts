"use client";

import { BLOCK_NODE_TYPE, NETWORK_NODE_TYPE } from "@/lib/features/diagram/blocksDocument";
import { MarkerType, type DefaultEdgeOptions, type EdgeTypes, type NodeTypes } from "@xyflow/react";
import type { CSSProperties } from "react";
import BlockNode from "./BlockNode";
import FloatingEdge, { FLOATING_EDGE_TYPE } from "./FloatingEdge";
import NetworkBox from "./NetworkBox";

// What both of B's canvases share: the editor's and the phone's read-only view.

export const nodeTypes: NodeTypes = { [BLOCK_NODE_TYPE]: BlockNode, [NETWORK_NODE_TYPE]: NetworkBox };
export const edgeTypes: EdgeTypes = { [FLOATING_EDGE_TYPE]: FloatingEdge };

// Applied to every edge as it is drawn: saved edges carry no type or marker.
// The arrow shows which way an edge reads ("Firewall to VM"). The invisible
// stroke a pointer grabs is 24 px wide (WCAG 2.5.8; React Flow's default is 20).
export const defaultEdgeOptions: DefaultEdgeOptions = {
  type: FLOATING_EDGE_TYPE,
  markerEnd: { type: MarkerType.ArrowClosed, width: 18, height: 18 },
  interactionWidth: 24,
};

// React Flow's own colours, from the theme's tokens, so the canvas follows
// the light and dark themes and its selection box matches the site's.
export const canvasStyle = {
  "--xy-background-color": "var(--background)",
  "--xy-edge-stroke": "color-mix(in oklab, var(--muted-foreground) 70%, transparent)",
  "--xy-edge-stroke-selected": "var(--primary)",
  "--xy-connectionline-stroke": "var(--primary)",
  "--xy-selection-background-color": "color-mix(in oklab, var(--primary) 6%, transparent)",
  "--xy-selection-border": "1px dashed var(--primary)",
  "--xy-controls-button-background-color": "var(--card)",
  "--xy-controls-button-background-color-hover": "var(--accent)",
  "--xy-controls-button-color": "var(--foreground)",
  "--xy-controls-button-color-hover": "var(--foreground)",
  "--xy-controls-button-border-color": "var(--border)",
  "--xy-controls-box-shadow": "0 1px 2px rgb(0 0 0 / 0.08)",
  "--xy-attribution-background-color": "color-mix(in oklab, var(--background) 70%, transparent)",
} as CSSProperties;
