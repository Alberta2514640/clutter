"use client";

import { ListChecks } from "lucide-react";

export default function Runs() {
  return (
    <main className="flex flex-col items-center justify-center h-full p-8 text-center text-sm text-muted-foreground">
      <ListChecks className="h-8 w-8 mb-3 text-primary/70" />
      <h1 className="text-lg font-semibold text-foreground mb-1">Runs</h1>
      <p>Workspace run history will appear here once you start a plan or apply.</p>
    </main>
  );
}
