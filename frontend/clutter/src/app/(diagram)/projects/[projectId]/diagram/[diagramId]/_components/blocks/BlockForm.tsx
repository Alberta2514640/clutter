"use client";

import CloudLogo from "@/components/common/CloudLogo";
import { Button } from "@/components/ui/button";
import { kindBadgeClass } from "@/lib/features/catalog/display";
import { openAdminPorts, openToEveryoneWarning } from "@/lib/features/catalog/fieldRules";
import { blockFieldError, blockFields, connectedBuilders, defaultValues, fieldShown } from "@/lib/features/catalog/forms";
import type { CatalogEntry } from "@/lib/features/catalog/types";
import type { BlockIssues } from "@/lib/features/deployments/compatibility";
import { VM_KIND, type BlockNode } from "@/lib/features/diagram/blocksDocument";
import { blockNameExample, RESOURCE_NAME_INPUT_MAX_LENGTH, resourceNameError } from "@/lib/features/diagram/resourceName";
import { cn } from "@/lib/utils";
import { Info, ShieldAlert, Trash2, TriangleAlert } from "lucide-react";
import FieldInput, { inputClass } from "./FieldInput";
import FunctionCodeUpload from "./FunctionCodeUpload";
import KindIcon from "./KindIcon";

type Props = {
  node: BlockNode;
  catalog: CatalogEntry[];
  title: string;
  // For a function's code upload: the caller's token, the diagram, and its
  // organization (or why that cannot be found).
  token: string | null;
  organizationId: string | null;
  organizationError: string | null;
  diagramId: string;
  // The connectors the organization has a connection to.
  connected: ReadonlySet<string>;
  // What the compatibility check says about this block, as last saved.
  issues: BlockIssues;
  // The block as last saved, or null when it is new: a check's problem with a
  // value shows only while the value is the one that was checked.
  saved: { name: string; fields: Record<string, unknown> } | null;
  // On a network: whether a VM that is not public runs in it.
  hasVM: boolean;
  onChange: (patch: { name?: string; fields?: Record<string, unknown> }) => void;
  onRemove: () => void;
};

const same = (a: unknown, b: unknown) => JSON.stringify(a ?? null) === JSON.stringify(b ?? null);

// blockNameProblem is why a block's name cannot be saved, or null. The
// editor's cap is stricter than B's (64); a longer name is said, not cut.
export const blockNameProblem = (kind: string, name: string) =>
  name.trim() === ""
    ? "Every block needs a name."
    : name.length > RESOURCE_NAME_INPUT_MAX_LENGTH
      ? `Use at most ${RESOURCE_NAME_INPUT_MAX_LENGTH} characters.`
      : resourceNameError(name, blockNameExample(kind));

// The blocks a cloud knows by their name, so a rename builds a new one.
const RENAME_REPLACES: Record<string, string> = {
  bucket: "Renaming a deployed bucket replaces it, and its files are deleted.",
  table: "Renaming a deployed table replaces it, and its items are deleted.",
};

// The selected block's name and its fields, from the manifests, with what the
// compatibility check found under the field it is about. Per-cloud settings
// (an image, an instance type) belong to each deployment instead.
export default function BlockForm({
  node,
  catalog,
  title,
  token,
  organizationId,
  organizationError,
  diagramId,
  connected,
  issues,
  saved,
  hasVM,
  onChange,
  onRemove,
}: Props) {
  const kind = node.data.kind;
  const allFields = blockFields(catalog, kind);
  const fields = allFields.filter((f) => fieldShown(f, { hasVM }));
  const nameError = blockNameProblem(kind, node.data.name);
  // What each connected cloud that builds this block calls it, and adds about it.
  const builtBy = connectedBuilders(catalog, kind, connected);
  // A check's problems with a field, while the field still holds the checked value.
  const checked = (field: string, value: unknown, savedValue: unknown) => (saved && same(value, savedValue) ? (issues.fields.get(field) ?? []) : []);
  const nameIssues = checked("name", node.data.name, saved?.name);
  const openWarning = kind === "firewall" ? openToEveryoneWarning(openAdminPorts(node.data.fields, defaultValues(allFields))) : null;

  return (
    <div className="space-y-5">
      <div className="flex items-start gap-3">
        <span className={cn(kindBadgeClass(kind), "flex size-9 shrink-0 items-center justify-center rounded-lg")}>
          <KindIcon kind={kind} className="size-4.5" />
        </span>
        <div className="min-w-0">
          <h3 className="text-sm font-semibold">{title}</h3>
          {builtBy.length > 0 ? (
            <ul aria-label="Built by" className="mt-1 space-y-1.5">
              {builtBy.map(({ cloud, service, help }) => (
                <li key={cloud.connector} className="text-muted-foreground text-xs">
                  <span className="flex items-center gap-2">
                    <CloudLogo connector={cloud.connector} shortName={cloud.shortName} src={cloud.logoSrc} size={16} />
                    <span>
                      <span className="text-foreground">{cloud.title}</span>: {service}
                    </span>
                  </span>
                  {help.length > 0 && (
                    <ul aria-label={`${service} on ${cloud.shortName}`} className="mt-1 space-y-0.5 pl-6 text-[11px] leading-snug">
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
            <p className="text-muted-foreground mt-1 text-xs">
              {connected.size === 0 ? "Connect a cloud to build this block." : "None of your connected clouds builds this block."}
            </p>
          )}
        </div>
      </div>
      {(issues.general.length > 0 || issues.notes.length > 0) && (
        <div data-testid="block-issues" className="space-y-1.5">
          {issues.general.map((p) => (
            <p key={p} className="text-warning flex items-start gap-1.5 text-xs">
              <TriangleAlert className="mt-px size-3.5 shrink-0" aria-hidden />
              {p}
            </p>
          ))}
          {issues.notes.map((n) => (
            <p key={n} data-testid="block-note" className="text-info flex items-start gap-1.5 text-xs">
              <Info className="mt-px size-3.5 shrink-0" aria-hidden />
              {n}
            </p>
          ))}
        </div>
      )}
      <div className="space-y-1.5">
        <label htmlFor={`block-name-${node.id}`} className="block text-xs font-medium">
          Name
        </label>
        <input
          id={`block-name-${node.id}`}
          value={node.data.name}
          spellCheck={false}
          aria-invalid={!!nameError || nameIssues.length > 0 || undefined}
          aria-describedby={nameError ? `block-name-${node.id}-error` : undefined}
          onChange={(e) => onChange({ name: e.target.value })}
          className={cn(inputClass, "font-mono")}
        />
        {nameError && (
          <p id={`block-name-${node.id}-error`} className="text-danger text-xs">
            {nameError}
          </p>
        )}
        {!nameError && <CheckedProblems messages={nameIssues} />}
        {RENAME_REPLACES[kind] && <p className="text-muted-foreground text-[11px]">{RENAME_REPLACES[kind]}</p>}
      </div>
      {fields.map((f) => {
        // B's message for a value the form already refuses is said once, by the field.
        const own = blockFieldError(kind, f, node.data.fields[f.name]);
        const problems = checked(f.name, node.data.fields[f.name], saved?.fields[f.name]).filter((m) => m !== own);
        return (
          <div key={f.name} className="space-y-1">
            <FieldInput
              id={`block-${node.id}-${f.name}`}
              field={f}
              kind={kind}
              value={node.data.fields[f.name]}
              invalid={problems.length > 0}
              onChange={(v) => {
                const next = { ...node.data.fields };
                if (v === undefined) delete next[f.name];
                else next[f.name] = v;
                onChange({ fields: next });
              }}
              after={
                f.name === "source_cidr" && openWarning ? (
                  <p data-testid="open-warning-form" className="border-warning/30 bg-warning/10 text-warning flex items-start gap-1.5 rounded-md border px-2 py-1.5 text-xs">
                    <ShieldAlert className="mt-px size-3.5 shrink-0" aria-hidden />
                    {openWarning}
                  </p>
                ) : kind === VM_KIND && f.name === "size" ? (
                  <p className="text-muted-foreground text-[11px]">Each cloud turns small, medium or large into its own server type. Preview shows the exact one.</p>
                ) : null
              }
            />
            <CheckedProblems messages={problems} />
          </div>
        );
      })}
      {kind === "function" && (
        <FunctionCodeUpload nodeId={node.id} token={token} organizationId={organizationId} organizationError={organizationError} diagramId={diagramId} />
      )}
      <div className="border-t pt-4">
        <Button type="button" variant="outline" size="sm" onClick={onRemove} className="text-danger hover:text-danger">
          <Trash2 aria-hidden />
          Remove block
        </Button>
        <p className="text-muted-foreground mt-1.5 text-xs">Or press Delete. You can undo it.</p>
      </div>
    </div>
  );
}

// What the compatibility check found wrong with one value, said under it.
function CheckedProblems({ messages }: { messages: string[] }) {
  if (messages.length === 0) return null;
  return (
    <>
      {messages.map((m) => (
        <p key={m} data-testid="field-problem" className="text-danger text-xs">
          {m}
        </p>
      ))}
    </>
  );
}
