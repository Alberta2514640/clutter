"use client";

import { Briefcase } from "lucide-react";

export default function Workspace() {
  return (
    <main className="flex flex-col items-center justify-center h-full p-8 text-center text-sm text-muted-foreground">
      <Briefcase className="h-8 w-8 mb-3 text-primary/70" />
      <h1 className="text-lg font-semibold text-foreground mb-1">Workspaces</h1>
      <p>Manage and deploy your project’s workspaces here.</p>
    </main>
  );
}
