"use client";

import { Button } from "@/components/ui/button";
import Link from "next/link";

export default function ProjectsPage() {
  return (
    <div className="p-10">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold text-white">Projects</h1>
          <p className="text-gray-400 mt-1">Select a project or create a new one.</p>
        </div>

        <Button asChild className="bg-teal-600 hover:bg-teal-500 text-white">
          <Link href="/projects/new">Create Project</Link>
        </Button>
      </div>

      <div className="rounded-xl border border-slate-700/50 bg-slate-800/40 p-6 text-gray-300">Your projects list will go here.</div>
    </div>
  );
}
