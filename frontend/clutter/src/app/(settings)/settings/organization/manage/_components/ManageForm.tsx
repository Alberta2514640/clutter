"use client";

import { organizationDescriptionHint, organizationDescriptionProblem } from "@/lib/features/organization/rules";
import React, { useEffect, useMemo, useState } from "react";

export interface ManageFormValues {
  orgName: string;
  orgId: string;
  description: string;
}

export interface ManageFormProps {
  initialValues: ManageFormValues;
  onSubmit: (values: ManageFormValues) => void | Promise<void>;
  onDelete?: () => void | Promise<void>;
  isSaving?: boolean;
  isDeleting?: boolean;
}

export default function ManageForm({ initialValues, onSubmit, onDelete, isSaving, isDeleting }: ManageFormProps) {
  const [values, setValues] = useState<ManageFormValues>(initialValues);

  useEffect(() => {
    setValues(initialValues);
  }, [initialValues]);

  const isDirty = useMemo(() => {
    const a = (initialValues.description ?? "").trim();
    const b = (values.description ?? "").trim();
    return a !== b;
  }, [initialValues.description, values.description]);

  const descriptionProblem = organizationDescriptionProblem(values.description);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!isDirty || descriptionProblem) return;
    await onSubmit({ ...values, description: values.description.trim() });
  };

  const saveDisabled = !!isSaving || !isDirty || !!descriptionProblem;

  return (
    <div>
      <form className="space-y-6" onSubmit={handleSubmit}>
        {/* Name + ID (read-only) */}
        <div className="grid gap-4 md:grid-cols-2">
          <div className="space-y-1.5">
            <label htmlFor="org-name" className="block text-sm font-medium text-slate-200">
              Organization name
            </label>
            <input
              id="org-name"
              type="text"
              value={values.orgName}
              disabled
              aria-describedby="org-name-hint"
              className="w-full rounded-lg bg-slate-950/70 border border-slate-800 px-3 py-2 text-sm text-slate-400 cursor-not-allowed"
            />
            <p id="org-name-hint" className="text-xs text-slate-500">
              The name cannot be changed.
            </p>
          </div>

          <div className="space-y-1.5">
            <label htmlFor="org-id" className="block text-sm font-medium text-slate-200">
              Organization ID
            </label>
            <input
              id="org-id"
              type="text"
              value={values.orgId}
              readOnly
              aria-describedby="org-id-hint"
              className="w-full rounded-lg bg-slate-950/70 border border-slate-800 px-3 py-2 text-sm text-slate-400"
            />
            <p id="org-id-hint" className="text-xs text-slate-500">
              This ID never changes. Give it when you ask for help.
            </p>
          </div>
        </div>

        {/* Description (editable) */}
        <div className="space-y-1.5">
          <label htmlFor="org-description" className="block text-sm font-medium text-slate-200">
            Description
          </label>
          <textarea
            id="org-description"
            value={values.description}
            onChange={(e) => setValues((prev) => ({ ...prev, description: e.target.value }))}
            rows={4}
            aria-describedby="org-description-hint"
            aria-invalid={!!descriptionProblem}
            className="w-full rounded-lg bg-slate-950/70 border border-slate-800 px-3 py-2 text-sm text-slate-50 focus:outline-none focus:ring-2 focus:ring-teal-500 focus:border-teal-500"
            placeholder="What is this organization for?"
          />
          <p id="org-description-hint" className={descriptionProblem ? "text-xs text-red-300" : "text-xs text-slate-500"}>
            {descriptionProblem ?? organizationDescriptionHint}
          </p>
        </div>

        {/* Save button */}
        <div className="flex items-center gap-3">
          <button type="submit" disabled={saveDisabled} className="px-4 py-2 rounded-lg bg-teal-600 text-sm font-medium text-white hover:bg-teal-500 disabled:opacity-60 transition">
            {isSaving ? "Saving…" : "Save changes"}
          </button>

          {!isDirty && <span className="text-xs text-slate-500">No changes to save</span>}
        </div>
      </form>

      {/* Danger zone */}
      <div className="mt-8 border-t border-slate-800 pt-4">
        <h3 className="text-sm font-semibold text-red-400 mb-2">Danger zone</h3>
        <p className="text-xs text-slate-400 mb-3 max-w-md">
          Deleting this organization removes its projects and diagrams. Clutter refuses while any deployment is still live, so destroy those first. This cannot be undone.
        </p>
        <button
          type="button"
          onClick={() => void onDelete?.()}
          disabled={!!isDeleting}
          className="px-4 py-2 rounded-lg border border-red-800 text-sm font-medium text-red-300 hover:bg-red-950 hover:text-red-200 disabled:opacity-60 transition">
          {isDeleting ? "Deleting…" : "Delete organization"}
        </button>
      </div>
    </div>
  );
}
