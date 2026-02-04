"use client";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useDeleteProject, useUpdateProject } from "@/lib/features/projects/hooks";
import { useProjectSettingsDraft } from "@/lib/features/projects/uiStore";
import { Layers, Loader2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useMemo } from "react";
import { useProjectContext } from "../_contexts/ProjectContext";

export default function ProjectSettingsPage() {
  const router = useRouter();
  const { token, projectId, project: currentProject } = useProjectContext();

  // Mutations
  const updateM = useUpdateProject(token);
  const deleteM = useDeleteProject(token);

  // Draft store
  const draft = useProjectSettingsDraft((s) => s.draft);
  const startDraft = useProjectSettingsDraft((s) => s.startDraft);
  const setName = useProjectSettingsDraft((s) => s.setName);
  const setDescription = useProjectSettingsDraft((s) => s.setDescription);

  // Initialize draft when project loads
  useEffect(() => {
    if (!currentProject) return;
    if (draft.projectId === currentProject.id) return;

    startDraft(
      currentProject.id,
      currentProject.name,
      currentProject.description || ""
    );
  }, [currentProject?.id, draft.projectId, startDraft]);

  const isSaving = updateM.isPending || deleteM.isPending;

  const error =
    (updateM.isError ? updateM.error : null) ||
    (deleteM.isError ? deleteM.error : null);

  const hasChanges = useMemo(() => {
    if (!currentProject) return false;
    return (
      draft.name !== currentProject.name ||
      draft.description !== (currentProject.description || "")
    );
  }, [draft.name, draft.description, currentProject]);

  const handleSave = async () => {
    if (!currentProject || !projectId || !hasChanges) return;

    await updateM.mutateAsync({
      projectId,
      data: {
        organizationId: currentProject.organizationId,
        name: draft.name.trim(),
        description: draft.description.trim(),
      },
    });
  };

  const handleCancel = () => {
    if (!currentProject) return;
    startDraft(
      currentProject.id,
      currentProject.name,
      currentProject.description || ""
    );
  };

  const handleDelete = async () => {
    if (!projectId || !currentProject) return;

    const confirmed = window.confirm(
      `Are you sure you want to delete "${currentProject.name}"? This action cannot be undone.`
    );

    if (confirmed) {
      await deleteM.mutateAsync({
        projectId: currentProject.id,
        organizationId: currentProject.organizationId,
      });
      router.push("/dashboard");
    }
  };

  if (!currentProject) {
    return null; // Layout handles loading state
  }

  const isDraftForCurrentProject = draft.projectId === currentProject.id;

  return (
    <div className="max-w-4xl mx-auto py-12 space-y-12 text-white">
      {error && (
        <Alert className="bg-red-900/20 border-red-800">
          <AlertDescription>{String(error)}</AlertDescription>
        </Alert>
      )}

      {/* Project name */}
      <div className="space-y-3">
        <Label className="text-gray-300">Icon and name</Label>
        <div className="flex items-center gap-4">
          <Button
            variant="outline"
            className="w-12 h-12 rounded-xl bg-slate-800 border-slate-700 text-gray-300 hover:border-teal-400"
          >
            <Layers className="w-5 h-5" />
          </Button>

          <Input
            className="bg-slate-800 border-slate-700 text-white"
            value={isDraftForCurrentProject ? draft.name : ""}
            onChange={(e) => setName(e.target.value)}
            disabled={isSaving}
            placeholder="Project name"
          />
        </div>
      </div>

      {/* Project description */}
      <div className="space-y-3">
        <Label className="text-gray-300">Description</Label>
        <Textarea
          className="bg-slate-800 border-slate-700 text-white"
          rows={3}
          value={isDraftForCurrentProject ? draft.description : ""}
          onChange={(e) => setDescription(e.target.value)}
          disabled={isSaving}
          placeholder="Describe your project..."
        />
      </div>

      {/* Save/Cancel buttons */}
      {hasChanges && (
        <div className="flex gap-3">
          <Button
            onClick={handleSave}
            disabled={isSaving || !draft.name.trim()}
            className="bg-teal-500 hover:bg-teal-600 text-white"
          >
            {isSaving ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                Saving...
              </>
            ) : (
              "Save Changes"
            )}
          </Button>

          <Button
            onClick={handleCancel}
            variant="outline"
            disabled={isSaving}
            className="border-slate-700 text-gray-300 hover:bg-slate-800"
          >
            Cancel
          </Button>
        </div>
      )}

      {/* Danger zone */}
      <div className="space-y-4 pt-8 border-t border-slate-800">
        <h3 className="text-lg font-semibold text-red-400">Danger zone</h3>
        <p className="text-gray-400 text-sm">
          When deleting a project, all associated workflows, credentials, and
          variables will be permanently removed.
        </p>
        <Button
          onClick={handleDelete}
          disabled={isSaving}
          className="bg-red-600 hover:bg-red-700 text-white"
        >
          Delete this project
        </Button>
      </div>
    </div>
  );
}