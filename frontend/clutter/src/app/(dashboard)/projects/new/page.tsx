"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

import { useCurrentOrganization } from "@/lib/features/organization/hooks";
import { useCreateProject } from "@/lib/features/projects/hooks";
import { useMe } from "@/lib/features/user/hooks";

export default function NewProjectPage() {
  const router = useRouter();
  const meQ = useMe();
  const user = meQ.data ?? null;
  const token = user?.token ?? null;

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [errorText, setErrorText] = useState<string | null>(null);

  const currentOrg = useCurrentOrganization(token);
  const { organizationId, organization } = currentOrg;

  const createProjectM = useCreateProject(token);
  const submitting = createProjectM.isPending;
  const canSubmit = !!name.trim() && !!token && !!organizationId && !submitting;

  const onCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorText(null);

    const trimmedName = name.trim();
    if (!trimmedName) return setErrorText("Enter a project name.");
    if (!token || !organizationId) return setErrorText("Your organization is still loading. Try again in a moment.");

    const desc = description.trim();
    const payload = {
      organizationId,
      name: trimmedName,
      ...(desc ? { description: desc } : {}), // omit when empty (backend rejects "")
    };

    try {
      const created = await createProjectM.mutateAsync(payload);
      router.replace(`/projects/${created.id}/diagrams`);
    } catch {
      setErrorText("Clutter could not create the project. Try again.");
    }
  };

  return (
    <div className="max-w-xl mx-auto px-6 py-12">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-white">Create New Project</h1>
        <p className="text-gray-400 mt-2">A project holds your diagrams.</p>
        {organization && (
          <p className="text-sm text-gray-500 mt-1">
            In <span className="text-gray-300">{organization.name}</span>. Change the organization in the sidebar.
          </p>
        )}
      </div>

      {currentOrg.isError ? (
        <p role="alert" className="text-sm text-red-300 bg-red-500/10 border border-red-500/20 rounded-lg p-3">
          Clutter could not load your organizations. Reload the page to try again.
        </p>
      ) : !currentOrg.isLoading && !organization ? (
        <p className="text-sm text-gray-300 bg-slate-800/50 border border-slate-700/50 rounded-lg p-3">
          You need an organization first.{" "}
          <Link href="/onboarding/create-org" className="text-teal-300 underline underline-offset-2">
            Create one
          </Link>
          .
        </p>
      ) : (
        <Card className="bg-slate-800/50 border-slate-700/50">
          <CardContent className="p-6">
            <form onSubmit={onCreate} className="space-y-5">
              <div className="space-y-2">
                <label htmlFor="project-name" className="text-sm text-gray-300">
                  Project name
                </label>
                <input
                  id="project-name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="e.g., Web App Infrastructure"
                  className="w-full rounded-lg bg-slate-900/60 border border-slate-700 px-3 py-2 text-white outline-none focus:border-teal-500/60"
                />
              </div>

              <div className="space-y-2">
                <label htmlFor="project-description" className="text-sm text-gray-300">
                  Description (optional)
                </label>
                <textarea
                  id="project-description"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="What is this project for?"
                  rows={4}
                  className="w-full rounded-lg bg-slate-900/60 border border-slate-700 px-3 py-2 text-white outline-none focus:border-teal-500/60"
                />
              </div>

              {errorText && (
                <p role="alert" className="text-sm text-red-300 bg-red-500/10 border border-red-500/20 rounded-lg p-3">
                  {errorText}
                </p>
              )}

              <Button type="submit" disabled={!canSubmit} className="w-full bg-teal-600 hover:bg-teal-500 text-white">
                {submitting ? "Creating..." : "Create Project"}
              </Button>

              <button type="button" onClick={() => router.push("/dashboard")} className="w-full text-sm text-gray-400 hover:text-gray-200">
                Back to Dashboard
              </button>
            </form>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
