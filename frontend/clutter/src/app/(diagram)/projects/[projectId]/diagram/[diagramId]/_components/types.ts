export type NodeType = "enclosure" | "shape" | "custom" | "dynamodb" | "s3" | "lambda" | "apigw";

export interface NodeInstance {
  id: string;
  type: NodeType;
  x: number;
  y: number;
}

export interface Edge {
  id: string;
  fromId: string;
  toId: string;
}

export type ActiveDrag =
  | null
  | { kind: "palette"; type: NodeType }
  | { kind: "node"; id: string };

export type DragData =
  | { kind: "palette"; nodeType: NodeType }
  | { kind: "node" };

export type PaletteGroup = "General" | "Storage" | "Compute" | "Network";

export type PaletteItem = { label: string; type: NodeType; group: PaletteGroup };
