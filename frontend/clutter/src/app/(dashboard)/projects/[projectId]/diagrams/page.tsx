"use client";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { useCreateDiagram, useDeleteDiagram } from "@/lib/features/diagram/hooks";
import { FileText, LayoutTemplate, Trash2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useProjectContext } from "../_contexts/ProjectContext";

export default function ProjectDiagramsPage() {
  const router = useRouter();
  const { projectId, token, user, project } = useProjectContext();

  // Create diagram mutation
  const createDiagramM = useCreateDiagram(token);
  const [createError, setCreateError] = useState<string | null>(null);

  // Delete diagram mutation
  const deleteDiagramM = useDeleteDiagram(token);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const handleCreateFromScratch = async () => {
    if (!token || !projectId || !project) return;

    setCreateError(null);

    try {
      const created = await createDiagramM.mutateAsync({
        organizationId: project.organizationId,
        projectId,
        name: "New Diagram",
      });

      // Go to the actual created diagram URL
      router.push(`/projects/${projectId}/diagram/${created.id}`);
    } catch (e) {
      setCreateError(String(e));
    }
  };

  const handleOpenDiagram = (diagramId: string) => {
    router.push(`/projects/${projectId}/diagram/${diagramId}`);
  };

  const handleDeleteDiagram = async (
    e: React.MouseEvent,
    diagramId: string,
    diagramName: string
  ) => {
    // Stop propagation to prevent opening the diagram
    e.stopPropagation();

    // Confirm deletion
    const confirmed = window.confirm(
      `Are you sure you want to delete "${diagramName}"?\n\nThis action cannot be undone.`
    );

    if (!confirmed || !project ) return;

    setDeleteError(null);

    try {
      await deleteDiagramM.mutateAsync({
        organizationId: project.organizationId,
        projectId,
        diagramId,
      });
    } catch (e) {
      setDeleteError(`Failed to delete ${diagramName}: ${String(e)}`);
    }
  };

  const firstName = user?.displayName?.split(" ")[0] || "there";

  // Use diagrams from project context instead of separate API call
  const diagrams = project?.diagrams ?? [];
  const hasDiagrams = diagrams.length > 0;

  console.log(diagrams)

  // Loading state (project is loading in context)
  if (!project) {
    return (
      <div className="max-w-5xl mx-auto px-6 py-12">
        <div className="text-center">
          <div className="inline-flex items-center justify-center w-16 h-16 mb-4 rounded-2xl bg-gradient-to-br from-teal-500 to-blue-600">
            <span className="text-3xl">👋</span>
          </div>
          <h2 className="text-3xl font-bold text-white mb-2">Welcome {firstName}!</h2>
          <p className="text-gray-400 text-lg">Loading your workspace...</p>
        </div>
      </div>
    );
  }

  // Empty state - Show create options
  if (!hasDiagrams) {
    return (
      <div className="max-w-5xl mx-auto px-6 py-12">
        {/* Welcome header */}
        <div className="text-center mb-12">
          <div className="inline-flex items-center justify-center w-16 h-16 mb-4 rounded-2xl bg-gradient-to-br from-teal-500 to-blue-600">
            <span className="text-3xl">👋</span>
          </div>

          <h2 className="text-3xl font-bold text-white mb-2">Welcome {firstName}!</h2>
          <p className="text-gray-400 text-lg">Create your first diagram to get started</p>

          {createError && (
            <p className="mt-4 text-sm text-red-400">Failed to create diagram: {createError}</p>
          )}
        </div>

        {/* Create new diagram options */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <Card
            onClick={createDiagramM.isPending ? undefined : handleCreateFromScratch}
            className={`
              bg-slate-800/50 border-slate-700/50 
              hover:border-teal-500/50 transition-all 
              ${createDiagramM.isPending ? "cursor-not-allowed opacity-60" : "cursor-pointer"}
              group
            `}
          >
            <CardContent className="p-8 text-center">
              <div className="inline-flex items-center justify-center w-16 h-16 mb-6 rounded-xl bg-slate-700/50 group-hover:bg-teal-500/20 transition-colors">
                <FileText className="w-8 h-8 text-gray-400 group-hover:text-teal-400 transition-colors" />
              </div>

              <h3 className="text-lg font-semibold text-white mb-2">
                {createDiagramM.isPending ? "Creating..." : "Start from scratch"}
              </h3>

              <p className="text-sm text-gray-400">Create a new workflow diagram</p>
            </CardContent>
          </Card>

          <Card className="bg-slate-800/50 border-slate-700/50 hover:border-teal-500/50 transition-all cursor-not-allowed group opacity-60">
            <CardContent className="p-8 text-center">
              <div className="inline-flex items-center justify-center w-16 h-16 mb-6 rounded-xl bg-slate-700/50 group-hover:bg-teal-500/20 transition-colors">
                <LayoutTemplate className="w-8 h-8 text-gray-400 group-hover:text-teal-400 transition-colors" />
              </div>

              <h3 className="text-lg font-semibold text-white mb-2">Start with a template</h3>
              <p className="text-sm text-gray-400">Coming soon</p>
            </CardContent>
          </Card>
        </div>
      </div>
    );
  }

  // Has diagrams - Show list
  return (
    <div className="max-w-5xl mx-auto px-6 py-12">
      {/* Error messages */}
      {createError && (
        <div className="mb-6 p-4 rounded-lg bg-red-900/20 border border-red-800">
          <p className="text-sm text-red-400">{createError}</p>
        </div>
      )}

      {deleteError && (
        <div className="mb-6 p-4 rounded-lg bg-red-900/20 border border-red-800">
          <p className="text-sm text-red-400">{deleteError}</p>
        </div>
      )}

      {/* Diagrams grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {diagrams.map((diagram) => {
          const isDeleting = deleteDiagramM.isPending && deleteDiagramM.variables?.diagramId === diagram.id;

          return (
            <Card
              key={diagram.id}
              onClick={() => handleOpenDiagram(diagram.id)}
              className={`
                bg-slate-900/40 border-slate-800 
                hover:border-teal-500/40 transition 
                ${isDeleting ? "opacity-50 cursor-not-allowed" : "cursor-pointer"}
                group relative
              `}
            >
              <CardContent className="p-5">
                <div className="flex items-start justify-between gap-4 mb-3">
                  <div className="w-12 h-12 flex items-center justify-center rounded-lg bg-slate-800 group-hover:bg-teal-500/20 transition-colors">
                    <FileText className="w-6 h-6 text-gray-400 group-hover:text-teal-400 transition-colors" />
                  </div>

                  {/* Delete button */}
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={(e) => handleDeleteDiagram(e, diagram.id, diagram.name)}
                    disabled={isDeleting}
                    className="
                      opacity-0 group-hover:opacity-100 
                      transition-opacity
                      h-8 w-8 p-0
                      hover:bg-red-900/30 
                      hover:text-red-400
                      text-gray-500
                    "
                    title="Delete diagram"
                  >
                    <Trash2 className="w-4 h-4" />
                  </Button>
                </div>

                <div className="min-w-0">
                  <div className="truncate text-white font-semibold mb-1">
                    {diagram.name}
                  </div>
                </div>

                {/* Deleting overlay */}
                {isDeleting && (
                  <div className="absolute inset-0 flex items-center justify-center bg-slate-900/60 rounded-lg">
                    <p className="text-sm text-gray-300">Deleting...</p>
                  </div>
                )}
              </CardContent>
            </Card>
          );
        })}
      </div>
    </div>
  );
}