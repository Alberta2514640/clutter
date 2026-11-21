"use client";

import type { LinkCard } from "@/lib/features/catalog/links";
import type { FieldSpec } from "@/lib/features/catalog/types";
import type { NodeLook } from "@/lib/features/deployments/views";
import { createContext, useContext } from "react";

// How a block reacts while a line is drawn: a valid target gets a ring, an invalid
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
  // A block's own warning that does not stop a save (a firewall open to
  // everyone), or null.
  warning: (id: string) => string | null;
  // Where the selected deployment's last apply built this block ("Live on
  // lab"), or null.
  live: (id: string) => string | null;
  // Whether that deployment is live but its last apply did not build this
  // block (one drawn since).
  notBuilt: (id: string) => boolean;
  // The one plain line under a block for people new to clouds, or null.
  hint: (kind: string) => string | null;
  // A block kind's fields, from the manifests.
  fieldsOf: (kind: string) => FieldSpec[];
  // Whether a VM that is not public runs in this network.
  hasVM: (id: string) => boolean;
  // Sets one field of a block, from the box's own switches.
  setField: (id: string, name: string, value: unknown) => void;
  edgeCard: (source: string, target: string) => EdgeCard | null;
  // Off on the phone's read-only view.
  editable: boolean;
};

export const CanvasContext = createContext<Canvas>({
  look: (block) => ({ subtitle: block.kind, iconSrc: null, dimmed: false, status: null, statusDetail: null, badges: [], viewCloud: null }),
  dropTarget: null,
  linkState: () => null,
  problems: () => [],
  warning: () => null,
  live: () => null,
  notBuilt: () => false,
  hint: () => null,
  fieldsOf: () => [],
  hasVM: () => false,
  setField: () => {},
  edgeCard: () => null,
  editable: true,
});

export const useCanvas = () => useContext(CanvasContext);
