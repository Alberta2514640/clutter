"use client";

import { fieldControl, fieldError, parseInput } from "@/lib/features/catalog/forms";
import type { FieldSpec } from "@/lib/features/catalog/types";
import HelpTip from "./HelpTip";

export const inputClass =
  "w-full rounded-md border border-slate-700 bg-slate-950/60 px-2.5 py-1.5 text-sm text-white outline-none focus:border-teal-500/70 disabled:opacity-50";

// FieldInput draws one manifest field and reports the parsed value. An unset
// value shows the field's default as a placeholder. The field's help, from
// the manifest, sits behind a "?" beside the label.
export default function FieldInput({
  id,
  field,
  value,
  onChange,
  disabled,
}: {
  id: string;
  field: FieldSpec;
  value: unknown;
  onChange: (value: unknown) => void;
  disabled?: boolean;
}) {
  const control = fieldControl(field);
  const error = fieldError(field, value);
  const hint = field.default !== undefined && field.default !== null ? `Default: ${String(field.default)}` : undefined;
  const help = field.help;

  const tip = help ? <HelpTip text={help} label={`About ${field.title}`} align="end" /> : null;

  return (
    <div className="space-y-1">
      {control.control === "checkbox" ? (
        <div className="flex items-center gap-2">
          <label htmlFor={id} className="flex items-center gap-2 text-sm text-slate-200">
            <input
              id={id}
              type="checkbox"
              disabled={disabled}
              checked={value === undefined ? field.default === true : value === true}
              onChange={(e) => onChange(parseInput(field, e.target.checked))}
              className="h-4 w-4 accent-teal-500"
            />
            {field.title}
          </label>
          {tip}
        </div>
      ) : (
        <>
          <div className="flex items-center gap-2">
            <label htmlFor={id} className="block text-xs font-medium text-slate-300">
              {field.title}
              {field.required ? "" : " (optional)"}
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
              aria-invalid={!!error}
              onChange={(e) => onChange(parseInput(field, e.target.value))}
              className={`${inputClass} resize-y font-mono text-xs leading-5 ${error ? "border-red-600" : ""}`}
            />
          ) : control.control === "select" ? (
            <select
              id={id}
              disabled={disabled}
              value={typeof value === "string" ? value : typeof field.default === "string" ? field.default : ""}
              onChange={(e) => onChange(parseInput(field, e.target.value))}
              className={inputClass}>
              {value === undefined && field.default === undefined && <option value="">Choose…</option>}
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
              aria-invalid={!!error}
              onChange={(e) => onChange(parseInput(field, e.target.value))}
              className={`${inputClass} ${error ? "border-red-600" : ""}`}
            />
          )}
        </>
      )}
      {error && <p className="text-xs text-red-400">{error}</p>}
    </div>
  );
}
