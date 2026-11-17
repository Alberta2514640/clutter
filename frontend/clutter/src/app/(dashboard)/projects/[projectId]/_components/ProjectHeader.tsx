"use client";

import { Button } from "@/components/ui/button";
import { Layers, Loader2, Plus } from "lucide-react";
import { useState } from "react";
import { useProjectContext } from "../_contexts/ProjectContext";
import NewDiagramDialog from "./NewDiagramDialog";

export function ProjectHeader() {
  const { project, projectLoading } = useProjectContext();
  const [choosing, setChoosing] = useState(false);

  const projectName = project?.name || (projectLoading ? "Loading..." : "Project not found");

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
            onClick={() => setChoosing(true)}
            disabled={projectLoading || !project}
            className="bg-teal-600 px-4 text-white hover:bg-teal-500 disabled:opacity-50"
          >
            <Plus className="w-4 h-4 mr-2" aria-hidden="true" />
            New diagram
          </Button>
        </div>
      </header>

      {choosing && <NewDiagramDialog onClose={() => setChoosing(false)} />}
    </>
  );
}
