"use client";

import { Button } from "@/components/ui/button";
import { usePreferencesStore } from "@/lib/features/preferences/uiStore";
import { findLastDiagram } from "@/lib/features/projects/lastDiagram";
import type { Project } from "@/lib/features/projects/types";
import { timeAgo } from "@/lib/features/projects/timeAgo";
import { useIsClient } from "@/lib/features/user/useIsClient";
import { ArrowRight, PenTool } from "lucide-react";
import Link from "next/link";
import { useState } from "react";

// "Continue where you left off": the diagram last opened in this team, kept
// in this browser. It shows only while that diagram is still in one of the
// team's projects.
export default function ContinueCard({ organizationId, projects }: { organizationId: string; projects: Project[] }) {
  const isClient = useIsClient();
  const last = usePreferencesStore((s) => s.lastDiagram[organizationId]);
  // Read once per render; the label needs no ticking.
  const [now] = useState(() => new Date());
  if (!isClient || !last) return null;

  const found = findLastDiagram(last, projects);
  if (!found) return null;
  const { project, diagram } = found;

  const opened = timeAgo(last.openedAt, now);
  const href = `/projects/${project.id}/diagram/${diagram.id}`;

  return (
    <section aria-labelledby="continue-heading" className="animate-fade-in mb-8">
      <h2 id="continue-heading" className="micro-label mb-3">
        Continue where you left off
      </h2>
      <div className="surface-card surface-card-interactive group relative flex flex-col gap-4 p-4 sm:flex-row sm:items-center">
        <span className="bg-primary/10 text-link flex size-11 shrink-0 items-center justify-center rounded-lg">
          <PenTool className="size-5" aria-hidden="true" />
        </span>
        <div className="min-w-0 flex-1">
          <p className="text-foreground truncate font-semibold">{diagram.name}</p>
          <p className="text-muted-foreground truncate text-sm">
            In {project.name}
            {opened && <> · Opened {opened}</>}
          </p>
        </div>
        <Button asChild variant="outline" className="sm:shrink-0">
          <Link href={href} className="after:absolute after:inset-0 after:rounded-xl">
            Open diagram
            <ArrowRight aria-hidden="true" />
          </Link>
        </Button>
      </div>
    </section>
  );
}
