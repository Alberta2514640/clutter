"use client";

import CloudMark from "@/components/clouds/CloudMark";
import { blockLook, cloudLook } from "@/lib/features/catalog/display";
import { blockFields } from "@/lib/features/catalog/forms";
import type { CatalogEntry } from "@/lib/features/catalog/types";
import type { BlockNode } from "@/lib/features/diagram/blocksDocument";
import { RESOURCE_NAME_INPUT_MAX_LENGTH, resourceNameError } from "@/lib/features/diagram/resourceName";
import { Trash2 } from "lucide-react";
import FieldInput, { inputClass } from "./FieldInput";
import FunctionCodeUpload from "./FunctionCodeUpload";

// A line under a field's label, for fields whose title alone does not say
// what they do. The fields themselves come from the manifests.
const FIELD_HELP: Record<string, string> = {
  // connectors.FieldStartupScript: cloud-init runs it on every cloud.
  startup_script: "Runs once as root when the VM first starts. Use it to install and start your app.",
};

type Props = {
  node: BlockNode;
  catalog: CatalogEntry[];
  title: string;
  disabled: boolean;
  onChange: (patch: { name?: string; fields?: Record<string, unknown> }) => void;
  onRemove: () => void;
};

// The selected block's name and its fields, from the manifests. Per-cloud
// settings (an image, an instance type) belong to each deployment instead.
export default function BlockForm({ node, catalog, title, disabled, onChange, onRemove }: Props) {
  const fields = blockFields(catalog, node.data.kind);
  const nameError = node.data.name.trim() === "" ? "Every block needs a name." : resourceNameError(node.data.name);
  // What each cloud that builds this block calls it.
  const builtBy = catalog.flatMap(({ manifest }) => {
    const look = blockLook(manifest, node.data.kind, title);
    return look.offered ? [{ cloud: cloudLook(manifest), service: look.name }] : [];
  });

  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-sm font-semibold text-white">{title}</h3>
        {builtBy.length > 0 ? (
          <ul aria-label="Built by" className="mt-1.5 space-y-1">
            {builtBy.map(({ cloud, service }) => (
              <li key={cloud.connector} className="flex items-center gap-2 text-xs text-slate-400">
                <CloudMark cloud={cloud} size={16} />
                <span>
                  <span className="text-slate-300">{cloud.title}</span>: {service}
                </span>
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-xs text-slate-500">No cloud Clutter offers builds this block.</p>
        )}
      </div>
      <div className="space-y-1">
        <label htmlFor={`block-name-${node.id}`} className="block text-xs font-medium text-slate-300">
          Name
        </label>
        <input
          id={`block-name-${node.id}`}
          value={node.data.name}
          maxLength={RESOURCE_NAME_INPUT_MAX_LENGTH}
          disabled={disabled}
          spellCheck={false}
          aria-invalid={!!nameError}
          onChange={(e) => onChange({ name: e.target.value })}
          className={`${inputClass} font-mono ${nameError ? "border-red-600" : ""}`}
        />
        {nameError && <p className="text-xs text-red-400">{nameError}</p>}
      </div>
      {fields.map((f) => (
        <FieldInput
          key={f.name}
          id={`block-${node.id}-${f.name}`}
          field={f}
          value={node.data.fields[f.name]}
          disabled={disabled}
          help={FIELD_HELP[f.name]}
          onChange={(v) => {
            const next = { ...node.data.fields };
            if (v === undefined) delete next[f.name];
            else next[f.name] = v;
            onChange({ fields: next });
          }}
        />
      ))}
      {node.data.kind === "function" && <FunctionCodeUpload nodeId={node.id} disabled={disabled} />}
      <button
        type="button"
        disabled={disabled}
        onClick={onRemove}
        className="inline-flex items-center gap-1.5 rounded-md border border-red-900 px-2.5 py-1.5 text-xs text-red-300 hover:bg-red-950 disabled:opacity-50">
        <Trash2 className="h-3.5 w-3.5" />
        Remove block
      </button>
    </div>
  );
}
