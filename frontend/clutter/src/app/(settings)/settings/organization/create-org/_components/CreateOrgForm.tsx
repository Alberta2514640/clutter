"use client";

import {
  organizationDescriptionHint,
  organizationDescriptionProblem,
  organizationNameCreateHint,
  organizationNamePlaceholder,
  organizationNameProblem,
} from "@/lib/features/organization/rules";
import { useState } from "react";

// The fields the backend takes for a new organization (POST /organization).
export interface CreateOrgFormValues {
  orgName: string;
  description: string;
}

export interface CreateOrgFormProps {
  onSubmit: (values: CreateOrgFormValues) => void | Promise<void>;
  onCancel: () => void;
  isCreating: boolean;
  error: string | null;
}

export default function CreateOrgForm({ onSubmit, onCancel, isCreating, error }: CreateOrgFormProps) {
  const [values, setValues] = useState<CreateOrgFormValues>({ orgName: "", description: "" });

  const nameProblem = organizationNameProblem(values.orgName);
  const descriptionProblem = organizationDescriptionProblem(values.description);
  const canSubmit = !nameProblem && !descriptionProblem && !isCreating;

  const handleChange = (field: keyof CreateOrgFormValues) => (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    setValues((prev) => ({ ...prev, [field]: e.target.value }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;
    await onSubmit({ orgName: values.orgName.trim(), description: values.description.trim() });
  };

  return (
    <div className="pb-12">
      <header className="py-6 md:px-6 md:py-10">
        <h1 className="mb-1 text-3xl font-semibold">New organization</h1>
        <p className="max-w-xl text-sm text-slate-400">
          Use another organization to keep work apart, for example for a client or a team. Each one has its own projects and connections.
        </p>
      </header>
    <section className="bg-slate-900/70 border border-slate-800 rounded-2xl shadow-xl">
      <form className="px-6 py-6 space-y-6" onSubmit={handleSubmit}>
        {error && (
          <p role="alert" className="rounded-lg border border-red-800 bg-red-900/20 p-3 text-sm text-red-300">
            {error}
          </p>
        )}

        <div className="grid gap-4 md:grid-cols-2">
          <div className="space-y-1.5">
            <label htmlFor="new-org-name" className="block text-sm font-medium text-slate-200">
              Organization name
            </label>
            <input
              id="new-org-name"
              type="text"
              required
              placeholder={organizationNamePlaceholder}
              value={values.orgName}
              onChange={handleChange("orgName")}
              aria-describedby="new-org-name-hint"
              className="w-full rounded-lg bg-slate-950/70 border border-slate-800 px-3 py-2 text-sm text-slate-50 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-teal-500 focus:border-teal-500"
            />
            <p id="new-org-name-hint" className="text-xs text-slate-500">
              {organizationNameCreateHint}
            </p>
          </div>

          <div className="space-y-1.5">
            <label htmlFor="new-org-description" className="block text-sm font-medium text-slate-200">
              Description (optional)
            </label>
            <textarea
              id="new-org-description"
              rows={3}
              placeholder="What is this organization for?"
              value={values.description}
              onChange={handleChange("description")}
              aria-describedby="new-org-description-hint"
              aria-invalid={!!descriptionProblem}
              className="w-full rounded-lg bg-slate-950/70 border border-slate-800 px-3 py-2 text-sm text-slate-50 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-teal-500 focus:border-teal-500"
            />
            <p id="new-org-description-hint" className={descriptionProblem ? "text-xs text-red-300" : "text-xs text-slate-500"}>
              {descriptionProblem ?? organizationDescriptionHint}
            </p>
          </div>
        </div>

        <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-800">
          <button type="button" onClick={onCancel} className="px-4 py-2 rounded-lg border border-slate-700 text-sm text-slate-300 hover:bg-slate-900/80 transition">
            Cancel
          </button>
          <button
            type="submit"
            disabled={!canSubmit}
            className="px-4 py-2 rounded-lg bg-teal-500 text-sm font-medium text-black hover:bg-teal-400 disabled:opacity-60 transition shadow-md shadow-teal-500/30">
            {isCreating ? "Creating…" : "Create organization"}
          </button>
        </div>
      </form>
    </section>
    </div>
  );
}
