"use client";

import type { LinkCard } from "@/lib/features/catalog/links";
import type { FieldSpec } from "@/lib/features/catalog/types";
import type { NodeLook } from "@/lib/features/deployments/views";
import { createContext, useContext } from "react";

// How a block reacts while a line is drawn: a valid target glows, an invalid
// one dims, and the block the line starts from does neither (null).
export type LinkState = "valid" | "invalid" | null;

// What a line's hover card shows: both ends, what the line means and why.
export type EdgeCard = LinkCard & { sourceName: string; targetName: string };

// What B's canvas hands its nodes and lines, so node data holds only what
// the document saves.
export type Canvas = {
  // How a block looks in the current "View as" choice.
  look: (block: { id: string; kind: string }) => NodeLook;
  // The box a VM being dragged would drop into, or null.
  dropTarget: string | null;
  linkState: (id: string) => LinkState;
  // The problems of a block, for its badge.
  problems: (id: string) => string[];
  // A block kind's fields, from the manifests.
  fieldsOf: (kind: string) => FieldSpec[];
  // Sets one field of a block, from the box's own switches.
  setField: (id: string, name: string, value: unknown) => void;
  edgeCard: (source: string, target: string) => EdgeCard | null;
};

export const CanvasContext = createContext<Canvas>({
  look: (block) => ({ subtitle: block.kind, iconSrc: null, dimmed: false, status: null, statusDetail: null, badges: [] }),
  dropTarget: null,
  linkState: () => null,
  problems: () => [],
  fieldsOf: () => [],
  setField: () => {},
  edgeCard: () => null,
});

export const useCanvas = () => useContext(CanvasContext);
