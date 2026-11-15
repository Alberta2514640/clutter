"use client";

import FirstSteps from "@/components/common/FirstSteps";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import type { Project } from "@/lib/features/projects/types";
import { FolderOpen, Plus } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";

interface ProjectsSectionProps {
  projects: Project[];
  isLoading: boolean;
  isError: boolean;
  onRetry: () => void;
}

export default function ProjectsSection({ projects, isLoading, isError, onRetry }: ProjectsSectionProps) {
  const router = useRouter();

  return (
    <div className="mb-8">
      <h2 className="text-2xl font-bold mb-4 flex items-center gap-2">
        <FolderOpen className="w-6 h-6 text-teal-400" aria-hidden="true" />
        Your Projects
      </h2>

      {isLoading ? (
        <Card className="bg-slate-900/50 backdrop-blur-xl border-slate-800/50">
          <CardContent role="status" className="py-12 text-center text-gray-400">
            Loading projects...
          </CardContent>
        </Card>
      ) : isError ? (
        <Card className="bg-slate-900/50 backdrop-blur-xl border-slate-800/50">
          <CardContent className="flex flex-col items-center gap-4 py-12 text-center">
            <p className="text-red-300">Clutter could not load your projects.</p>
            <Button variant="outline" onClick={onRetry} className="border-slate-700 bg-slate-900 text-slate-100 hover:bg-slate-800">
              Try again
            </Button>
          </CardContent>
        </Card>
      ) : projects.length === 0 ? (
        <Card className="bg-slate-900/50 backdrop-blur-xl border-slate-800/50">
          <CardContent className="grid gap-8 py-10 md:grid-cols-[1fr_1.2fr] md:items-center">
            <div className="flex flex-col items-center text-center">
              <FolderOpen className="w-16 h-16 text-gray-600 mb-4" aria-hidden="true" />
              <h3 className="text-xl font-semibold text-white mb-2">No projects yet</h3>
              <p className="text-gray-400 mb-6">Follow these steps to your first deploy.</p>
              <Button onClick={() => router.push("/projects/new")} className="bg-teal-500 hover:bg-teal-600">
                <Plus className="w-4 h-4 mr-2" aria-hidden="true" />
                Create Project
              </Button>
            </div>
            <FirstSteps />
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {projects.map((project) => (
            <Link key={project.id} href={`/projects/${project.id}/diagrams`}>
              <Card className="bg-slate-900/50 backdrop-blur-xl border-slate-800/50 hover:border-teal-500/50 transition-all cursor-pointer group">
                <CardHeader>
                  <CardTitle className="text-white group-hover:text-teal-400 transition-colors">{project.name}</CardTitle>
                  <CardDescription className="text-gray-400 line-clamp-2">{project.description || "No description"}</CardDescription>
                </CardHeader>
              </Card>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
