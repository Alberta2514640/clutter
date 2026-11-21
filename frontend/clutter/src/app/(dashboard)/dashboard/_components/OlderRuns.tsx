"use client";

import ErrorState from "@/components/common/ErrorState";
import StatusDot, { type StatusTone } from "@/components/common/StatusDot";
import { Skeleton } from "@/components/ui/skeleton";
import { useHasOlderDiagrams } from "@/lib/features/diagram/hooks";
import { useProjectRecentActivity } from "@/lib/features/logs/hooks";
import type { Project } from "@/lib/features/projects/types";
import { timeAgo } from "@/lib/features/projects/timeAgo";
import Link from "next/link";
import { useMemo, useState } from "react";

// Runs of A's older AWS-only diagrams, which keep their history in A. It shows
// only when the team has such a diagram; building-block diagrams show their
// runs on the project's Deployments tab and the diagram's Deploy view.

const TONE: Record<string, StatusTone> = { SUCCESS: "success", FAILED: "danger", RUNNING: "primary", QUEUED: "info" };
const COMMAND: Record<string, string> = { plan: "Plan", apply: "Deploy", destroy: "Destroy" };
const statusLabel = (status: string) => status.charAt(0).toUpperCase() + status.slice(1).toLowerCase();

export default function OlderRuns({ token, organizationId, projects }: { token: string; organizationId: string; projects: Project[] }) {
  const projectIds = useMemo(() => projects.map((p) => p.id), [projects]);
  const older = useHasOlderDiagrams(token, projectIds);
  const activityQ = useProjectRecentActivity(token, organizationId, projects);
  const [now] = useState(() => new Date());
  const items = activityQ.data ?? [];
  // Nothing to act on until an older diagram has run.
  if (!older.hasOlder || (!activityQ.isLoading && !activityQ.isError && items.length === 0)) return null;

  return (
    <section aria-labelledby="older-runs-heading" className="mt-12">
      <h2 id="older-runs-heading" className="micro-label">
        Runs from the older AWS editor
      </h2>

      <div className="surface-card mt-3 overflow-hidden">
        {activityQ.isLoading ? (
          <div role="status" className="space-y-3 p-4">
            <span className="sr-only">Loading runs…</span>
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        ) : activityQ.isError ? (
          // The list asks again every 10 seconds by itself, so there is no Retry.
          <ErrorState variant="inline" className="p-4" description="These runs could not load. Clutter tries again in a few seconds." />
        ) : (
          <ul className="divide-y">
            {items.map((item) => (
              <li key={`${item.diagramId}-${item.commandId}`}>
                <Link
                  href={`/projects/${item.projectId}/diagram/${item.diagramId}`}
                  className="hover:bg-accent/60 focus-visible:bg-accent/60 flex min-w-0 flex-wrap items-center gap-x-4 gap-y-1 px-4 py-3 text-sm outline-none transition-colors">
                  <StatusDot tone={TONE[item.status.toUpperCase()] ?? "neutral"} label={statusLabel(item.status)} className="w-24 shrink-0" />
                  <span className="text-foreground min-w-0 flex-1 truncate">
                    {COMMAND[item.command.toLowerCase()] ?? item.command} · {item.diagramName}
                    <span className="text-muted-foreground"> in {item.projectName}</span>
                  </span>
                  <span className="text-muted-foreground shrink-0 text-xs">{timeAgo(item.createdAt, now)}</span>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </div>
    </section>
  );
}
