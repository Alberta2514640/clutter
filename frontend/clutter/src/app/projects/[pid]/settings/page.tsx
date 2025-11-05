"use client";

import { Settings as SettingsIcon } from "lucide-react";

export default function Settings() {
  return (
    <main className="flex flex-col items-center justify-center h-full p-8 text-center text-sm text-muted-foreground">
      <SettingsIcon className="h-8 w-8 mb-3 text-primary/70" />
      <h1 className="text-lg font-semibold mb-1 text-foreground">Project Settings</h1>
      <p>Project configuration options will appear here.</p>
    </main>
  );
}
