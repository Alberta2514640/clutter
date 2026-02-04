"use client";

import { Card, CardContent } from "@/components/ui/card";
import { useDiagrams } from "@/lib/features/diagram/hooks";
import { FileText, LayoutTemplate } from "lucide-react";
import { useRouter } from "next/navigation";
import { useProjectContext } from "../_contexts/ProjectContext";

export default function ProjectDiagramsPage() {
  const router = useRouter();
  const { projectId, token, user } = useProjectContext();
  
  // Fetch diagrams for this project
  const diagramsQ = useDiagrams(token, projectId);
  
  const handleCreateFromScratch = () => {
    router.push(`/projects/${projectId}/diagrams/newDiagram`);
  };
  
  const handleOpenDiagram = (diagramId: string) => {
    router.push(`/projects/${projectId}/diagram/${diagramId}`);
  };
  
  const firstName = user?.displayName?.split(" ")[0] || "there";
  
  return (
    <div className="max-w-5xl mx-auto px-6 py-12">
      {/* Welcome header */}
      <div className="text-center mb-12">
        <div className="inline-flex items-center justify-center w-16 h-16 mb-4 rounded-2xl bg-gradient-to-br from-teal-500 to-blue-600">
          <span className="text-3xl">👋</span>
        </div>
        
        <h2 className="text-3xl font-bold text-white mb-2">
          Welcome {firstName}!
        </h2>
        
        <p className="text-gray-400 text-lg">
          {diagramsQ.isLoading
            ? "Loading your workspace..."
            : "Pick a diagram or create a new one"}
        </p>
      </div>
      
      {/* Create new diagram options */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <Card
          onClick={handleCreateFromScratch}
          className="bg-slate-800/50 border-slate-700/50 hover:border-teal-500/50 transition-all cursor-pointer group"
        >
          <CardContent className="p-8 text-center">
            <div className="inline-flex items-center justify-center w-16 h-16 mb-6 rounded-xl bg-slate-700/50 group-hover:bg-teal-500/20 transition-colors">
              <FileText className="w-8 h-8 text-gray-400 group-hover:text-teal-400 transition-colors" />
            </div>
            <h3 className="text-lg font-semibold text-white mb-2">
              Start from scratch
            </h3>
            <p className="text-sm text-gray-400">
              Create a new workflow diagram
            </p>
          </CardContent>
        </Card>
        
        <Card className="bg-slate-800/50 border-slate-700/50 hover:border-teal-500/50 transition-all cursor-pointer group opacity-60">
          <CardContent className="p-8 text-center">
            <div className="inline-flex items-center justify-center w-16 h-16 mb-6 rounded-xl bg-slate-700/50 group-hover:bg-teal-500/20 transition-colors">
              <LayoutTemplate className="w-8 h-8 text-gray-400 group-hover:text-teal-400 transition-colors" />
            </div>
            <h3 className="text-lg font-semibold text-white mb-2">
              Start with a template
            </h3>
            <p className="text-sm text-gray-400">Coming soon</p>
          </CardContent>
        </Card>
      </div>
      
      {/* Existing diagrams */}
      <div className="mt-10">
        <h3 className="text-lg font-semibold text-white mb-4">
          Your diagrams
        </h3>
        
        {diagramsQ.isError ? (
          <p className="text-sm text-red-400">
            Failed to load diagrams: {String(diagramsQ.error)}
          </p>
        ) : diagramsQ.isLoading ? (
          <p className="text-sm text-gray-400">Loading diagrams…</p>
        ) : (diagramsQ.data?.length ?? 0) === 0 ? (
          <p className="text-sm text-gray-400">
            No diagrams yet — create your first one above.
          </p>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {diagramsQ.data!.map((diagram) => (
              <Card
                key={diagram.id}
                onClick={() => handleOpenDiagram(diagram.id)}
                className="bg-slate-900/40 border-slate-800 hover:border-teal-500/40 transition cursor-pointer"
              >
                <CardContent className="p-5">
                  <div className="flex items-start justify-between gap-4">
                    <div className="min-w-0">
                      <div className="truncate text-white font-semibold">
                        {diagram.name}
                      </div>
                      <div className="mt-1 text-xs text-gray-400 truncate">
                        {diagram.id}
                      </div>
                    </div>
                    
                    <div className="text-xs text-gray-400 shrink-0">
                      {diagram.uiLayout?.nodes?.length ?? 0} nodes
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}