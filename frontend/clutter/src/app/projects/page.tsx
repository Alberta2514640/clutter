"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";

export default function ProjectsPage() {
  return (
    <main className="p-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-3xl font-bold tracking-tight">Your Projects</h1>
        <Button asChild>
          <Link href="/projects/new">
            <Plus className="h-4 w-4 mr-2" />
            New Project
          </Link>
        </Button>
      </div>
      {/* <p className="text-muted-foreground">This is where your saved Clutter projects will appear.</p> */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {["My AWS Demo", "F1 Telemetry Infra", "Student Portal"].map((name, i) => (
          <Link key={i} href={`/projects/${i + 1}`} className="block border rounded-lg p-4 hover:bg-muted transition">
            <h2 className="font-semibold text-foreground mb-1">{name}</h2>
            <p className="text-xs text-muted-foreground">Last updated: 2 days ago</p>
          </Link>
        ))}
      </div>
    </main>
  );
}
