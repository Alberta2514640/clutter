import type { NodeType, PaletteGroup, PaletteItem } from "../types";

export const GRID_SIZE = 24;
export const NODE_WIDTH = 140;
export const NODE_HEIGHT = 44;

export const PALETTE_ITEMS: PaletteItem[] = [
  { label: "Enclosure", type: "enclosure", group: "General" },
  { label: "Shapes", type: "shape", group: "General" },
  { label: "Custom", type: "custom", group: "General" },
  { label: "DynamoDB", type: "dynamodb", group: "Storage" },
  { label: "S3", type: "s3", group: "Storage" },
  { label: "Lambda", type: "lambda", group: "Compute" },
  { label: "API Gateway", type: "apigw", group: "Network" },
];

export const GROUPS: PaletteGroup[] = ["General", "Storage", "Compute", "Network"];

export function snap(n: number) {
  return Math.round(n / GRID_SIZE) * GRID_SIZE;
}

export function clamp(n: number, min: number, max: number) {
  return Math.max(min, Math.min(max, n));
}

export function uid() {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) return crypto.randomUUID();
  return `id_${Math.random().toString(16).slice(2)}_${Date.now()}`;
}

export function renderNodeLabel(type: NodeType) {
  switch (type) {
    case "dynamodb":
      return "DynamoDB";
    case "s3":
      return "S3";
    case "lambda":
      return "Lambda";
    case "apigw":
      return "API Gateway";
    case "enclosure":
      return "Enclosure";
    case "shape":
      return "Shape";
    case "custom":
      return "Custom";
    default:
      return type;
  }
}