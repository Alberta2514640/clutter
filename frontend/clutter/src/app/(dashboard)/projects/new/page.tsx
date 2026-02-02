"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";

import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";

import { useMe } from "@/lib/features/user/hooks";
import { useCreateProject } from "@/lib/features/projects/hooks";
import { useOrganizations } from "@/lib/features/organization/hooks";

export default function NewProjectPage() {
  const router = useRouter();
  const meQ = useMe();
  const user = meQ.data ?? null;
  const token = user?.token ?? null;
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");

  useEffect(() => {
    if (!name && user?.displayName) {
      setName("");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.displayName]);

  const orgsQ = useOrganizations(token);
  const organizationId = useMemo(() => {
    return orgsQ.data?.[0]?.id ?? null;
  }, [orgsQ.data]);

  const createProjectM = useCreateProject();
  const [errorText, setErrorText] = useState<string | null>(null);
  const submitting = createProjectM.isPending;
  const canSubmit = !!name.trim() && !!token && !!organizationId && !submitting;

  const onCreate = () => {
    setErrorText(null);

    const trimmedName = name.trim();
    if (!trimmedName) {
      setErrorText("Project name is required.");
      return;
    }
    if (!token) {
      setErrorText("Missing auth token. Please log in again.");
      return;
    }
    if (!organizationId) {
      setErrorText("Missing organizationId. Please select an organization first.");
      return;
    }

    createProjectM.mutate(
      {
        token,
        organizationId,
        name: trimmedName,
        description: description.trim(),
      },
      {
        onSuccess: (created) => {
          router.replace(`/projects/${created.projectId}/diagrams`);
        },
        onError: (e) => {
          setErrorText(String(e));
        },
      },
    );
  };

  return (
    <div className="max-w-xl mx-auto px-6 py-12">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-white">Create New Project</h1>
        <p className="text-gray-400 mt-2">Create a project to start building diagrams, variables, and workspaces.</p>
      </div>

      <Card className="bg-slate-800/50 border-slate-700/50">
        <CardContent className="p-6 space-y-5">
          <div className="space-y-2">
            <label className="text-sm text-gray-300">Project name</label>
            <input value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g., Web App Infrastructure" className="w-full rounded-lg bg-slate-900/60 border border-slate-700 px-3 py-2 text-white outline-none focus:border-teal-500/60" />
          </div>

          <div className="space-y-2">
            <label className="text-sm text-gray-300">Description (optional)</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="What is this project for?"
              rows={4}
              className="w-full rounded-lg bg-slate-900/60 border border-slate-700 px-3 py-2 text-white outline-none focus:border-teal-500/60"
            />
          </div>

          {(errorText || createProjectM.isError) && <pre className="whitespace-pre-wrap text-sm text-red-300 bg-red-500/10 border border-red-500/20 rounded-lg p-3">{errorText ?? String(createProjectM.error?.message ?? createProjectM.error)}</pre>}

          <Button onClick={onCreate} disabled={!canSubmit} className="w-full bg-teal-600 hover:bg-teal-500 text-white">
            {submitting ? "Creating..." : "Create Project"}
          </Button>

          <button type="button" onClick={() => router.push("/projects")} className="w-full text-sm text-gray-400 hover:text-gray-200">
            Back to projects
          </button>

          {/* Debug helpers */}
          {/* <div className="pt-2 text-xs text-gray-500 space-y-1">
            <div>Auth status: {meQ.isLoading ? "Loading..." : meQ.isError ? "Error" : "OK"}</div>
            <div>token: {token ? "present" : "missing"}</div>
            <div>orgs: {orgsQ.isLoading ? "Loading..." : orgsQ.isError ? "Error" : `${orgsQ.data?.length ?? 0}`}</div>
            <div>organizationId: {organizationId ?? "missing"}</div>
          </div> */}
        </CardContent>
      </Card>
    </div>
  );
}
