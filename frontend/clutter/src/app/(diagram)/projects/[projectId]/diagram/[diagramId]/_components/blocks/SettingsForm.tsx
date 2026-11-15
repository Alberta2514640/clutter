"use client";

import { AdapterApiError, adapterErrorMessage } from "@/lib/adapterApi";
import { fieldError } from "@/lib/features/catalog/forms";
import type { FieldSpec } from "@/lib/features/catalog/types";
import { useConnectionOptions } from "@/lib/features/connections/hooks";
import { optionsState, settingView, TYPE_OWN } from "@/lib/features/connections/options";
import { Loader2 } from "lucide-react";
import { useState } from "react";
import FieldInput, { inputClass } from "./FieldInput";

// SettingsForm draws a deployment's settings. A setting that names something
// in the user's cloud (an image, a flavor, a network) is a list of the
// connection's own values when the cloud can list them; otherwise, or when
// the user chooses to type, it is the manifest's text box. The lists are
// asked for only while the form is open.
export default function SettingsForm({
  token,
  organizationId,
  connectionId,
  idPrefix,
  spec,
  values,
  onChange,
  disabled,
}: {
  token: string | null;
  organizationId: string;
  connectionId: string;
  idPrefix: string;
  spec: FieldSpec[];
  values: Record<string, unknown>;
  onChange: (name: string, value: unknown) => void;
  disabled?: boolean;
}) {
  const wantsLists = spec.some((f) => f.type === "string");
  const optionsQ = useConnectionOptions(token, organizationId, wantsLists ? connectionId : null);
  const state = optionsState({
    isLoading: optionsQ.isLoading,
    isError: optionsQ.isError,
    data: optionsQ.data,
    errorCode: optionsQ.error instanceof AdapterApiError ? optionsQ.error.code : undefined,
    errorMessage: optionsQ.isError ? adapterErrorMessage(optionsQ.error) : "",
  });
  // The settings the user chose to type rather than pick.
  const [typing, setTyping] = useState<Record<string, boolean>>({});

  return (
    <div className="space-y-2">
      {spec.map((f) => {
        const id = `${idPrefix}-${f.name}`;
        const view = settingView(f, state, values[f.name], !!typing[f.name]);
        if (view.control === "list") {
          const error = fieldError(f, values[f.name]);
          return (
            <div key={f.name} className="space-y-1">
              <label htmlFor={id} className="block text-xs font-medium text-slate-300">
                {f.title}
                {f.required ? "" : " (optional)"}
              </label>
              <select
                id={id}
                disabled={disabled}
                value={view.current}
                onChange={(e) => {
                  if (e.target.value === TYPE_OWN) setTyping((t) => ({ ...t, [f.name]: true }));
                  else onChange(f.name, e.target.value);
                }}
                className={inputClass}>
                {view.current === "" && <option value="">Choose…</option>}
                {view.choices.map((c) => (
                  <option key={c.value} value={c.value}>
                    {c.label}
                  </option>
                ))}
                <option value={TYPE_OWN}>Type another name…</option>
              </select>
              {error && <p className="text-xs text-red-400">{error}</p>}
            </div>
          );
        }
        return (
          <div key={f.name} className="space-y-1">
            <FieldInput id={id} field={f} value={values[f.name]} disabled={disabled} onChange={(v) => onChange(f.name, v)} />
            {view.loading ? (
              <p role="status" className="flex items-center gap-1.5 text-[11px] text-slate-400">
                <Loader2 className="h-3 w-3 animate-spin" aria-hidden />
                {view.note}
              </p>
            ) : view.note ? (
              <p className="text-[11px] text-amber-300">{view.note}</p>
            ) : null}
            {view.canChoose && (
              <button
                type="button"
                onClick={() => setTyping((t) => ({ ...t, [f.name]: false }))}
                className="text-[11px] text-teal-300 hover:underline">
                Choose from your cloud&apos;s list
              </button>
            )}
          </div>
        );
      })}
    </div>
  );
}
