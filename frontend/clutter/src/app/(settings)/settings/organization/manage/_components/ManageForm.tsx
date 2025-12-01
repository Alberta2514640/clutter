// ManageForm.tsx
"use client";

import { useState } from "react";

export interface ManageFormValues {
  orgName: string;
  slug: string;
  timeZone: string;
}

export interface ManageFormProps {
  initialValues: ManageFormValues;
  onSubmit: (values: ManageFormValues) => void | Promise<void>;
  onCancel?: () => void;
  onDelete?: () => void | Promise<void>;
  isSaving?: boolean;
  isDeleting?: boolean;
}

export default function ManageForm({ initialValues, onSubmit, onDelete, isDeleting, }: ManageFormProps) {
  const [values, setValues] = useState<ManageFormValues>(initialValues);

  const handleChange =
    (field: keyof ManageFormValues) =>
    (
      e: React.ChangeEvent<
        HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement
      >
    ) => {
      setValues((prev) => ({ ...prev, [field]: e.target.value }));
    };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await onSubmit(values);
  };

  const handleDelete = async () => {
    if (!onDelete) return;
    await onDelete();
  };

  return (
    <div>
      <form className="space-y-6" onSubmit={handleSubmit}>
        {/* Name + ID (read-only) */}
        <div className="grid gap-4 md:grid-cols-2">
          <div className="space-y-1.5">
            <label className="block text-sm font-medium text-slate-200">
              Organization name
            </label>
            <input
              type="text"
              value={values.orgName}
              disabled
              className="w-full rounded-lg bg-slate-950/70 border border-slate-800 px-3 py-2 text-sm text-slate-400 cursor-not-allowed"
            />
            <p className="text-xs text-slate-500">
              Organization name cannot be changed.
            </p>
          </div>

          <div className="space-y-1.5">
            <label className="block text-sm font-medium text-slate-200">
              Organization ID
            </label>
            <input
              type="text"
              value={values.slug}
              disabled
              className="w-full rounded-lg bg-slate-950/70 border border-slate-800 px-3 py-2 text-sm text-slate-400 cursor-not-allowed"
            />
            <p className="text-xs text-slate-500">
              Used in URLs and APIs. This ID is fixed.
            </p>
          </div>
        </div>
        <div className="grid gap-4">

          <div className="space-y-1.5">
            <label className="block text-sm font-medium text-slate-200">
              Time zone
            </label>
            <select
              value={values.timeZone}
              onChange={handleChange("timeZone")}
              className="w-full rounded-lg bg-slate-950/70 border border-slate-800 px-3 py-2 text-sm text-slate-50 focus:outline-none focus:ring-2 focus:ring-teal-500 focus:border-teal-500"
            >
              <option value="America/Edmonton">(UTC-07:00) Edmonton</option>
              <option value="America/Denver">(UTC-07:00) Denver</option>
              <option value="America/New_York">(UTC-05:00) New York</option>
              <option value="UTC">(UTC) Coordinated Universal Time</option>
            </select>
            <p className="text-xs text-slate-500">
              Used for run timestamps and schedules.
            </p>
          </div>
        </div>
      </form>

      {/* Danger zone */}
      <div className="mt-8 border-t border-slate-800 pt-4">
        <h3 className="text-sm font-semibold text-red-400 mb-2">
          Danger zone
        </h3>
        <p className="text-xs text-slate-400 mb-3 max-w-md">
          Deleting this organization will remove access to all projects,
          workspaces, and runs. This action cannot be undone.
        </p>
        <button
          type="button"
          onClick={handleDelete}
          disabled={isDeleting}
          className="px-4 py-2 rounded-lg bg-red-600 text-sm font-medium text-white hover:bg-red-500 disabled:opacity-60 transition"
        >
          {isDeleting ? "Deleting…" : "Delete organization"}
        </button>
      </div>
    </div>
  );
}