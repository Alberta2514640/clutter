"use client";

import ConfirmDialog from "@/components/common/ConfirmDialog";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useDeleteProject, useUpdateProject } from "@/lib/features/projects/hooks";
import { useProjectSettingsDraft } from "@/lib/features/projects/uiStore";
import { isConflict } from "@/lib/features/user/session";
import { Loader2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { useProjectContext } from "../_contexts/ProjectContext";

export default function ProjectSettingsPage() {
  const router = useRouter();
  const { token, projectId, project: currentProject } = useProjectContext();

  const updateM = useUpdateProject(token);
  const deleteM = useDeleteProject(token);

  const draft = useProjectSettingsDraft((s) => s.draft);
  const startDraft = useProjectSettingsDraft((s) => s.startDraft);
  const setName = useProjectSettingsDraft((s) => s.setName);
  const setDescription = useProjectSettingsDraft((s) => s.setDescription);

  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  useEffect(() => {
    if (!currentProject) return;
    if (draft.projectId === currentProject.id) return;
    startDraft(currentProject.id, currentProject.name, currentProject.description || "");
  }, [currentProject, draft.projectId, startDraft]);

  const isSaving = updateM.isPending || deleteM.isPending;

  const hasChanges = useMemo(() => {
    if (!currentProject) return false;
    return draft.name !== currentProject.name || draft.description !== (currentProject.description || "");
  }, [draft.name, draft.description, currentProject]);

  const handleSave = async () => {
    if (!currentProject || !projectId || !hasChanges) return;
    const name = draft.name.trim();
    const desc = draft.description.trim();
    try {
      await updateM.mutateAsync({
        projectId,
        data: {
          organizationId: currentProject.organizationId,
          name,
          ...(desc ? { description: desc } : {}),
        },
      });
    } catch {
      // Shown from updateM.isError below.
    }
  };

  const handleCancel = () => {
    if (!currentProject) return;
    startDraft(currentProject.id, currentProject.name, currentProject.description || "");
  };

  const handleDeleteConfirm = async () => {
    if (!projectId || !currentProject) return;
    setDeleteError(null);
    try {
      await deleteM.mutateAsync({
        projectId: currentProject.id,
        organizationId: currentProject.organizationId,
      });
    } catch (err) {
      setDeleteError(
        isConflict(err)
          ? "A diagram in this project still has live deployments. Destroy them on the diagram's Deploy tab, then delete the project."
          : "Clutter could not delete the project. Try again.",
      );
      return;
    }
    setShowDeleteModal(false);
    router.push("/dashboard");
  };

  if (!currentProject) return null;

  const isDraftForCurrentProject = draft.projectId === currentProject.id;

  return (
    <>
      {showDeleteModal && (
        <ConfirmDialog
          title="Delete project"
          confirmLabel="Delete project"
          pendingLabel="Deleting…"
          isPending={deleteM.isPending}
          error={deleteError}
          onCancel={() => setShowDeleteModal(false)}
          onConfirm={() => void handleDeleteConfirm()}>
          <p>
            Delete <span className="text-white font-medium">&quot;{currentProject.name}&quot;</span> and all its diagrams? This cannot be undone.
          </p>
          <p>If a diagram has live deployments, Clutter refuses until you destroy them.</p>
        </ConfirmDialog>
      )}

      <div className="max-w-4xl mx-auto py-12 space-y-12 text-white">
        {updateM.isError && (
          <Alert className="bg-red-900/20 border-red-800">
            <AlertDescription>Clutter could not save your changes. Try again.</AlertDescription>
          </Alert>
        )}

        {/* Project name */}
        <div className="space-y-3">
          <Label htmlFor="project-settings-name" className="text-gray-300">
            Name
          </Label>
          <Input
            id="project-settings-name"
            className="bg-slate-800 border-slate-700 text-white"
            value={isDraftForCurrentProject ? draft.name : ""}
            onChange={(e) => setName(e.target.value)}
            disabled={isSaving}
            placeholder="Project name"
          />
        </div>

        {/* Project description */}
        <div className="space-y-3">
          <Label htmlFor="project-settings-description" className="text-gray-300">
            Description
          </Label>
          <Textarea
            id="project-settings-description"
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
            <Button onClick={handleSave} disabled={isSaving || !draft.name.trim()} className="bg-teal-500 hover:bg-teal-600 text-white">
              {isSaving ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" aria-hidden="true" />
                  Saving...
                </>
              ) : (
                "Save Changes"
              )}
            </Button>
            <Button onClick={handleCancel} variant="outline" disabled={isSaving} className="border-slate-700 text-gray-300 hover:bg-slate-800">
              Cancel
            </Button>
          </div>
        )}

        {/* Danger zone */}
        <div className="space-y-4 pt-8 border-t border-slate-800">
          <h3 className="text-lg font-semibold text-red-400">Danger zone</h3>
          <p className="text-gray-400 text-sm">Deleting a project removes all its diagrams for good. Clutter refuses while any of them has live deployments.</p>
          <Button
            onClick={() => {
              setDeleteError(null);
              setShowDeleteModal(true);
            }}
            disabled={isSaving}
            variant="outline"
            className="border-red-800 bg-transparent text-red-300 hover:bg-red-950 hover:text-red-200">
            Delete this project
          </Button>
        </div>
      </div>
    </>
  );
}
