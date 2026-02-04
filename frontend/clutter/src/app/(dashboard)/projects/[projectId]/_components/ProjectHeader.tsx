"use client";

import { Button } from "@/components/ui/button";
import { Layers, Loader2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { useProjectContext } from "../_contexts/ProjectContext";

export function ProjectHeader() {
  const router = useRouter();
  const { project, projectLoading, projectId } = useProjectContext();
  
  const handleCreateDiagram = () => {
    router.push(`/projects/${projectId}/diagrams/newDiagram`);
  };
  
  const projectName = project?.name || (projectLoading ? "Loading..." : "Project not found");
  
  return (
    <header className="border-b border-slate-800 px-12 py-8 flex items-center justify-between">
      <div className="flex items-center gap-4">
        <div className="w-14 h-14 flex items-center justify-center rounded-2xl bg-gradient-to-br from-teal-500 to-blue-600 p-[2px]">
          <div className="w-full h-full flex items-center justify-center rounded-2xl bg-slate-900">
            {projectLoading ? (
              <Loader2 className="w-6 h-6 text-teal-300 animate-spin" />
            ) : (
              <Layers className="w-6 h-6 text-teal-300" />
            )}
          </div>
        </div>
        <h1 className="text-2xl font-semibold">{projectName}</h1>
      </div>
      
      <div className="flex items-center gap-2">
        <Button
          onClick={handleCreateDiagram}
          disabled={projectLoading || !project}
          className="bg-gradient-to-br from-teal-600 to-blue-600 hover:opacity-90 text-white px-4 disabled:opacity-50"
        >
          Create New Diagram
        </Button>
      </div>
    </header>
  );
}