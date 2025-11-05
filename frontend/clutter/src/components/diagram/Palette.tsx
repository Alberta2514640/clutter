"use client";

import React from "react";
import { useEditor } from "@/lib/state/editorStore";
import type { Node as DomainNode } from "@/lib/types";
import { logSuccess, logError } from "@/components/diagram/Console";

type TemplateKey = "Lambda" | "DynamoDB" | "S3" | "APIGateway";
type Spec = Record<string, unknown>;

const TEMPLATES: Record<
  TemplateKey,
  Omit<DomainNode, "nodeId" | "canvasId" | "ui" | "spec"> & {
    uiLabel?: string;
    spec: Spec;
  }
> = {
  Lambda: {
    resourceType: "Lambda",
    spec: { runtime: "nodejs20.x", memory: 256, timeout: 10 },
    iac: { moduleAlias: "aws_lambda_function", version: "~> 5.0.0" },
    uiLabel: "Lambda",
  },
  DynamoDB: {
    resourceType: "DynamoDB",
    spec: { billing_mode: "PAY_PER_REQUEST", hash_key: "pk" },
    iac: { moduleAlias: "aws_dynamodb_table", version: "~> 5.0.0" },
    uiLabel: "DynamoDB",
  },
  S3: {
    resourceType: "S3",
    spec: { versioning: { enabled: true }, acl: "private" },
    iac: { moduleAlias: "aws_s3_bucket", version: "~> 5.0.0" },
    uiLabel: "S3 Bucket",
  },
  APIGateway: {
    resourceType: "APIGateway",
    spec: { protocol: "HTTP", cors: true },
    iac: { moduleAlias: "aws_apigatewayv2_api", version: "~> 5.0.0" },
    uiLabel: "API Gateway",
  },
};

export default function Palette() {
  const { bundle, upsertNode } = useEditor();
  const canvasId = bundle?.canvas.canvasId;

  const addNode = (kind: TemplateKey) => {
    if (!canvasId) return;
    try {
      const t = TEMPLATES[kind];
      const node = makeNode({
        canvasId,
        resourceType: t.resourceType,
        spec: t.spec,
        iac: t.iac,
        uiLabel: t.uiLabel,
      });
      upsertNode(node);
      logSuccess("Node added", { nodeId: node.nodeId, resourceType: node.resourceType });
    } catch (e) {
      logError("Failed to add node");
    }
  };

  return (
    <div className="w-full md:w-56 border rounded-xl bg-neutral-950/90 text-neutral-100">
      <div className="px-4 py-3 border-b border-neutral-800">
        <div className="text-sm font-semibold text-neutral-300">Palette</div>
        <div className="text-xs text-neutral-500">
          {bundle ? bundle.canvas.name : "No canvas loaded"}
        </div>
      </div>

      <div className="p-3 grid grid-cols-2 md:grid-cols-1 gap-2">
        <PaletteButton label="Lambda" onClick={() => addNode("Lambda")} />
        <PaletteButton label="DynamoDB" onClick={() => addNode("DynamoDB")} />
        <PaletteButton label="S3" onClick={() => addNode("S3")} />
        <PaletteButton label="API Gateway" onClick={() => addNode("APIGateway")} />

        <div className="col-span-2 md:col-span-1 pt-2 border-t border-neutral-800 text-[11px] text-neutral-500">
          Tip: Add a node, then position it on the canvas. Edit details in the Inspector.
        </div>
      </div>
    </div>
  );
}

/* ---------------- helpers ---------------- */

function PaletteButton({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className="w-full text-left px-3 py-2 rounded border border-neutral-700 hover:border-neutral-500 bg-neutral-900/60 text-sm"
    >
      {label}
    </button>
  );
}

function makeNode(input: {
  canvasId: string;
  resourceType: string;
  spec: Spec;
  iac?: DomainNode["iac"];
  uiLabel?: string;
}: DomainNode): DomainNode {
  const id = genId(input.resourceType);
  const position = { x: 80 + Math.random() * 280, y: 80 + Math.random() * 220 };
  return {
    nodeId: id,
    canvasId: input.canvasId,
    resourceType: input.resourceType,
    spec: input.spec,
    iac: input.iac,
    ui: { label: input.uiLabel, position },
  };
}

function genId(prefix: string) {
  const slug = prefix.replace(/\W+/g, "").toLowerCase();
  return `${slug}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 6)}`;
}
