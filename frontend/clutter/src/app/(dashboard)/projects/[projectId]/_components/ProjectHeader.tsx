"use client";

import { Button } from "@/components/ui/button";
import { useCreateDiagram } from "@/lib/features/diagram/hooks";
import { Layers, Loader2, Plus } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useProjectContext } from "../_contexts/ProjectContext";

export function ProjectHeader() {
  const router = useRouter();
  const { project, projectLoading, projectId, token } = useProjectContext();

  // Create diagram mutation
  const createDiagramM = useCreateDiagram(token);
  const [createError, setCreateError] = useState<string | null>(null);

  /**
   * Generate a unique diagram name based on existing diagrams
   * Pattern: "New Diagram", "New Diagram 1", "New Diagram 2", etc.
   */
  const generateDiagramName = (): string => {
    const diagrams = project?.diagrams ?? [];
    
    if (diagrams.length === 0) {
      return "New Diagram";
    }

    // Find all existing "New Diagram" names
    const newDiagramNames = diagrams
      .map(d => d.name)
      .filter(name => name.match(/^New Diagram( \d+)?$/));

    if (newDiagramNames.length === 0) {
      return "New Diagram";
    }

    // Extract numbers from existing names
    const numbers = newDiagramNames
      .map(name => {
        const match = name.match(/^New Diagram(?: (\d+))?$/);
        if (!match) return 0;
        return match[1] ? parseInt(match[1], 10) : 0;
      })
      .filter(n => !isNaN(n));

    // Find the highest number
    const maxNumber = Math.max(0, ...numbers);
    
    // Return next number
    if (maxNumber === 0) {
      // We have "New Diagram" but no numbered ones yet
      return "New Diagram 1";
    }
    
    return `New Diagram ${maxNumber + 1}`;
  };

  const handleCreateDiagram = async () => {
    if (!token || !projectId || !project) return;

    setCreateError(null);

    try {
      const diagramName = generateDiagramName();
      
      const created = await createDiagramM.mutateAsync({
        organizationId: project.organizationId,
        projectId,
        name: diagramName,
      });

      // Navigate to the newly created diagram
      router.push(`/projects/${projectId}/diagram/${created.id}`);
    } catch {
      setCreateError("Clutter could not create the diagram. Try again.");
    }
  };

  const projectName = project?.name || (projectLoading ? "Loading..." : "Project not found");
  const isCreating = createDiagramM.isPending;

  return (
    <>
      <header className="flex flex-wrap items-center justify-between gap-4 border-b border-slate-800 px-4 py-6 md:px-12 md:py-8">
        <div className="flex min-w-0 items-center gap-4">
          <div className="w-14 h-14 flex items-center justify-center rounded-2xl bg-gradient-to-br from-teal-500 to-blue-600 p-[2px]">
            <div className="w-full h-full flex items-center justify-center rounded-2xl bg-slate-900">
              {projectLoading ? (
                <Loader2 className="w-6 h-6 text-teal-300 animate-spin" />
              ) : (
                <Layers className="w-6 h-6 text-teal-300" />
              )}
            </div>
          </div>
          <h1 className="truncate text-2xl font-semibold">{projectName}</h1>
        </div>

        <div className="flex items-center gap-2">
          <Button
            onClick={handleCreateDiagram}
            disabled={projectLoading || !project || isCreating}
            className="bg-teal-600 px-4 text-white hover:bg-teal-500 disabled:opacity-50"
          >
            {isCreating ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" aria-hidden="true" />
                Creating...
              </>
            ) : (
              <>
                <Plus className="w-4 h-4 mr-2" aria-hidden="true" />
                New diagram
              </>
            )}
          </Button>
        </div>
      </header>

      {/* Error banner */}
      {createError && (
        <div className="px-12 py-3 bg-red-900/20 border-b border-red-800">
          <p role="alert" className="text-sm text-red-400">{createError}</p>
        </div>
      )}
    </>
  );
}