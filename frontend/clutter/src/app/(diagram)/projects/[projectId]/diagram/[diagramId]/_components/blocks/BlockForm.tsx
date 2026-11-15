"use client";

import CloudMark from "@/components/clouds/CloudMark";
import { blockFields, connectedBuilders, fieldShown } from "@/lib/features/catalog/forms";
import type { CatalogEntry } from "@/lib/features/catalog/types";
import type { BlockIssues } from "@/lib/features/deployments/compatibility";
import type { BlockNode } from "@/lib/features/diagram/blocksDocument";
import { RESOURCE_NAME_INPUT_MAX_LENGTH, resourceNameError } from "@/lib/features/diagram/resourceName";
import { Info, Trash2, TriangleAlert } from "lucide-react";
import FieldInput, { inputClass } from "./FieldInput";
import FunctionCodeUpload from "./FunctionCodeUpload";

type Props = {
  node: BlockNode;
  catalog: CatalogEntry[];
  title: string;
  disabled: boolean;
  // The connectors the organization has a connection to.
  connected: ReadonlySet<string>;
  // What the compatibility check says about this block, as last saved.
  issues: BlockIssues;
  // The block as last saved, or null when it is new: a check's problem with a
  // value shows only while the value is the one that was checked.
  saved: { name: string; fields: Record<string, unknown> } | null;
  // On a network: whether a VM that is not public runs in it.
  hasPrivateVM: boolean;
  onChange: (patch: { name?: string; fields?: Record<string, unknown> }) => void;
  onRemove: () => void;
};

const same = (a: unknown, b: unknown) => JSON.stringify(a ?? null) === JSON.stringify(b ?? null);

// The selected block's name and its fields, from the manifests, with what the
// compatibility check found under the field it is about. Per-cloud settings
// (an image, an instance type) belong to each deployment instead.
export default function BlockForm({ node, catalog, title, disabled, connected, issues, saved, hasPrivateVM, onChange, onRemove }: Props) {
  const fields = blockFields(catalog, node.data.kind).filter((f) => fieldShown(f, { hasPrivateVM }));
  const nameError = node.data.name.trim() === "" ? "Every block needs a name." : resourceNameError(node.data.name);
  // What each connected cloud that builds this block calls it, and adds about it.
  const builtBy = connectedBuilders(catalog, node.data.kind, connected);
  // A check's problems with a field, while the field still holds the checked value.
  const checked = (field: string, value: unknown, savedValue: unknown) => (saved && same(value, savedValue) ? (issues.fields.get(field) ?? []) : []);
  const nameIssues = checked("name", node.data.name, saved?.name);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-sm font-semibold text-white">{title}</h3>
        {builtBy.length > 0 ? (
          <ul aria-label="Built by" className="mt-1.5 space-y-1.5">
            {builtBy.map(({ cloud, service, help }) => (
              <li key={cloud.connector} className="text-xs text-slate-400">
                <span className="flex items-center gap-2">
                  <CloudMark cloud={cloud} size={16} />
                  <span>
                    <span className="text-slate-300">{cloud.title}</span>: {service}
                  </span>
                </span>
                {help.length > 0 && (
                  <ul aria-label={`${service} on ${cloud.shortName}`} className="mt-1 space-y-0.5 pl-6 text-[11px] leading-snug text-slate-400">
                    {help.map((h) => (
                      <li key={h} className="break-words">
                        {h}
                      </li>
                    ))}
                  </ul>
                )}
              </li>
            ))}
          </ul>
        ) : (
          <p className="mt-1 text-xs text-slate-500">
            {connected.size === 0 ? "Connect a cloud to build this block." : "None of your connected clouds builds this block."}
          </p>
        )}
      </div>
      {(issues.general.length > 0 || issues.notes.length > 0) && (
        <div data-testid="block-issues" className="space-y-1">
          {issues.general.map((p) => (
            <p key={p} className="flex items-start gap-1.5 text-xs text-amber-300">
              <TriangleAlert className="mt-px h-3.5 w-3.5 shrink-0" aria-hidden />
              {p}
            </p>
          ))}
          {issues.notes.map((n) => (
            <p key={n} data-testid="block-note" className="flex items-start gap-1.5 text-xs text-sky-200">
              <Info className="mt-px h-3.5 w-3.5 shrink-0" aria-hidden />
              {n}
            </p>
          ))}
        </div>
      )}
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
          aria-invalid={!!nameError || nameIssues.length > 0}
          onChange={(e) => onChange({ name: e.target.value })}
          className={`${inputClass} font-mono ${nameError || nameIssues.length > 0 ? "border-red-600" : ""}`}
        />
        {nameError && <p className="text-xs text-red-400">{nameError}</p>}
        {!nameError && <CheckedProblems messages={nameIssues} />}
      </div>
      {fields.map((f) => {
        const problems = checked(f.name, node.data.fields[f.name], saved?.fields[f.name]);
        return (
          <div key={f.name} className="space-y-1">
            <FieldInput
              id={`block-${node.id}-${f.name}`}
              field={f}
              value={node.data.fields[f.name]}
              disabled={disabled}
              invalid={problems.length > 0}
              onChange={(v) => {
                const next = { ...node.data.fields };
                if (v === undefined) delete next[f.name];
                else next[f.name] = v;
                onChange({ fields: next });
              }}
            />
            <CheckedProblems messages={problems} />
          </div>
        );
      })}
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

// What the compatibility check found wrong with one value, said under it.
function CheckedProblems({ messages }: { messages: string[] }) {
  if (messages.length === 0) return null;
  return (
    <>
      {messages.map((m) => (
        <p key={m} data-testid="field-problem" className="text-xs text-red-400">
          {m}
        </p>
      ))}
    </>
  );
}
