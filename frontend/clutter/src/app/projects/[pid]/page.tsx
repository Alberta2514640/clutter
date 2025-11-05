"use client";

import { FolderKanban } from "lucide-react";

export default function ProjectOverview() {
  return (
    <main className="flex flex-col items-center justify-center h-full p-8 text-center text-sm text-muted-foreground">
      <FolderKanban className="h-8 w-8 mb-3 text-primary/70" />
      <h1 className="text-lg font-semibold text-foreground mb-1">Project Overview</h1>
      <p>General information and recent activity for this project will appear here.</p>
    </main>
  );
}
