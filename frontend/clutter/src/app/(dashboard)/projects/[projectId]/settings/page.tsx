"use client";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useProjectId } from "@/lib/hooks/useProjectId";
import { useProjectActions, useProjectState } from "@/lib/stores/projectStore";
import { Layers, Loader2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";

export default function ProjectSettingsPage() {
  const router = useRouter();
  const projectId = useProjectId();
  
  // Store state and actions
  const { currentProject, isSaving, error } = useProjectState();
  const { loadProject, updateProject, deleteProject } = useProjectActions();

  // Local form state - initialize with empty strings
  const [projectName, setProjectName] = useState("");
  const [description, setDescription] = useState("");

  // Load project data on mount
  useEffect(() => {
    if (projectId && (!currentProject || currentProject.projectId !== projectId)) {
      loadProject(projectId);
    }
  }, [projectId]); // Only depend on projectId

  // Sync form with store data when currentProject changes
  // This is acceptable because it only runs when external data (store) changes
  useEffect(() => {
    if (currentProject && currentProject.projectId === projectId) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setProjectName(currentProject.name);
      setDescription(currentProject.description || "");
    }
  }, [currentProject?.projectId, projectId]); // Only sync when project ID changes

  // Calculate hasChanges using useMemo instead of useEffect
  const hasChanges = useMemo(() => {
    if (!currentProject) return false;
    
    return (
      projectName !== currentProject.name ||
      description !== (currentProject.description || "")
    );
  }, [projectName, description, currentProject]);

  const handleSave = async () => {
    if (!projectId || !hasChanges) return;

    await updateProject(projectId, {
      name: projectName.trim(),
      description: description.trim(),
    });
  };

  const handleCancel = () => {
    if (currentProject) {
      setProjectName(currentProject.name);
      setDescription(currentProject.description || "");
    }
  };

  const handleDelete = async () => {
    if (!projectId) return;

    const confirmed = window.confirm(
      `Are you sure you want to delete "${currentProject?.name}"? This action cannot be undone.`
    );

    if (confirmed) {
      await deleteProject(projectId);
      router.push("/dashboard");
    }
  };

  if (!currentProject) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="w-8 h-8 animate-spin text-teal-400" />
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto py-12 space-y-12 text-white">
      {/* Error Alert */}
      {error && (
        <Alert className="bg-red-900/20 border-red-800">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {/* ICON + NAME FIELDSET */}
      <div className="space-y-3">
        <Label className="text-gray-300">Icon and name</Label>
        <div className="flex items-center gap-4">
          {/* Icon button */}
          <Button
            variant="outline"
            className="w-12 h-12 rounded-xl bg-slate-800 border-slate-700 text-gray-300 hover:border-teal-400"
          >
            <Layers className="w-5 h-5" />
          </Button>
          {/* Project name input */}
          <Input
            className="bg-slate-800 border-slate-700 text-white"
            value={projectName}
            onChange={(e) => setProjectName(e.target.value)}
            disabled={isSaving}
          />
        </div>
      </div>

      {/* DESCRIPTION FIELDSET */}
      <div className="space-y-3">
        <Label className="text-gray-300">Description</Label>
        <Textarea
          className="bg-slate-800 border-slate-700 text-white"
          rows={3}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          disabled={isSaving}
        />
      </div>

      {/* Save Button */}
      {hasChanges && (
        <div className="flex gap-3">
          <Button
            onClick={handleSave}
            disabled={isSaving || !projectName.trim()}
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

      {/* DANGER ZONE */}
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