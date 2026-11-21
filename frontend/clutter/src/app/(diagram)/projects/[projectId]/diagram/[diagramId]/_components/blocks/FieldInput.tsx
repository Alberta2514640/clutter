"use client";

import { blockFieldError, fieldControl, fieldError, fieldLabel, parseInput } from "@/lib/features/catalog/forms";
import type { FieldSpec } from "@/lib/features/catalog/types";
import { cn } from "@/lib/utils";
import { useId, type ReactNode } from "react";
import HelpTip from "./HelpTip";

export const inputClass =
  "border-input bg-background text-foreground placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-ring/50 h-9 w-full min-w-0 rounded-md border px-2.5 text-sm shadow-xs outline-none transition-[color,box-shadow,border-color] focus-visible:ring-[3px] disabled:cursor-not-allowed disabled:opacity-50 aria-invalid:border-danger aria-invalid:ring-danger/20";

// FieldInput draws one manifest field and reports the parsed value. An unset
// value shows the field's default as a placeholder. The field's help, from
// the manifest, sits behind a "?" beside the label. With a block kind, the
// value is also checked against the field's grammar (a firewall's allowed
// traffic, an address range) as it is typed.
export default function FieldInput({
  id,
  field,
  value,
  onChange,
  kind,
  disabled,
  invalid,
  after,
}: {
  id: string;
  field: FieldSpec;
  value: unknown;
  onChange: (value: unknown) => void;
  // The block kind the field belongs to, for its grammar.
  kind?: string;
  disabled?: boolean;
  // Set when a check outside this field (B's compatibility check) refuses
  // the value; the caller says why under the field.
  invalid?: boolean;
  // A note under the field, such as a warning that does not stop a save.
  after?: ReactNode;
}) {
  const control = fieldControl(field);
  const ownError = kind ? blockFieldError(kind, field, value) : fieldError(field, value);
  const error = !!ownError || !!invalid;
  const hint = field.default !== undefined && field.default !== null ? `Default: ${String(field.default)}` : undefined;
  const help = field.help;
  const errorId = useId();
  const exampleId = useId();
  // A title's own example ("for example tcp/22") shows under the field.
  const { label, hint: example } = fieldLabel(field.title);

  // The card spans the label row, so it stays inside the panel wherever the "?" is.
  const tip = help ? <HelpTip text={help} label={`About ${label}`} align="row" /> : null;
  const described = [ownError ? errorId : null, example ? exampleId : null].filter(Boolean).join(" ") || undefined;

  return (
    <div className="space-y-1.5">
      {control.control === "checkbox" ? (
        <div className="relative flex items-center gap-2">
          <label htmlFor={id} className="flex min-h-6 items-center gap-2 text-sm">
            <input
              id={id}
              type="checkbox"
              disabled={disabled}
              checked={value === undefined ? field.default === true : value === true}
              onChange={(e) => onChange(parseInput(field, e.target.checked))}
              className="accent-primary size-4"
            />
            {label}
          </label>
          {tip}
        </div>
      ) : (
        <>
          <div className="relative flex items-center gap-2">
            <label htmlFor={id} className="block text-xs font-medium">
              {label}
              {field.required ? "" : <span className="text-muted-foreground font-normal"> (optional)</span>}
            </label>
            {tip}
          </div>
          {control.control === "textarea" ? (
            <textarea
              id={id}
              rows={8}
              disabled={disabled}
              spellCheck={false}
              autoCapitalize="off"
              autoCorrect="off"
              wrap="off"
              value={typeof value === "string" ? value : ""}
              placeholder={hint}
              aria-invalid={error || undefined}
              aria-describedby={described}
              onChange={(e) => onChange(parseInput(field, e.target.value))}
              className={cn(inputClass, "h-auto resize-y py-2 font-mono text-xs leading-5")}
            />
          ) : control.control === "select" ? (
            // An optional list starts with "Default", which clears the value
            // so the field's default (or the cloud's) applies again.
            <select
              id={id}
              disabled={disabled}
              value={typeof value === "string" ? value : !field.required ? "" : typeof field.default === "string" ? field.default : ""}
              onChange={(e) => onChange(parseInput(field, e.target.value))}
              className={inputClass}>
              {!field.required ? (
                <option value="">{typeof field.default === "string" && field.default ? `Default (${field.default})` : "Default"}</option>
              ) : (
                value === undefined && field.default === undefined && <option value="">Choose…</option>
              )}
              {control.options.map((o) => (
                <option key={o} value={o}>
                  {o}
                </option>
              ))}
            </select>
          ) : (
            <input
              id={id}
              type={control.control === "number" ? "number" : "text"}
              step={control.control === "number" ? 1 : undefined}
              disabled={disabled}
              spellCheck={false}
              value={value === undefined || value === null ? "" : String(value)}
              placeholder={hint}
              aria-invalid={error || undefined}
              aria-describedby={described}
              onChange={(e) => onChange(parseInput(field, e.target.value))}
              className={cn(inputClass, kind && (field.name === "ingress" || field.name.endsWith("cidr")) && "font-mono")}
            />
          )}
        </>
      )}
      {example && (
        <p id={exampleId} className="text-muted-foreground font-mono text-[11px]">
          {example}
        </p>
      )}
      {ownError && (
        <p id={errorId} className="text-danger text-xs">
          {ownError}
        </p>
      )}
      {after}
    </div>
  );
}
