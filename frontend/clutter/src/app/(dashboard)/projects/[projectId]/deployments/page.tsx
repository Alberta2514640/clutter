"use client";

import IsoStack from "@/components/brand/IsoStack";
import CloudLogo from "@/components/common/CloudLogo";
import EmptyState from "@/components/common/EmptyState";
import ErrorState from "@/components/common/ErrorState";
import StatusDot from "@/components/common/StatusDot";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useOrgDeployments } from "@/lib/features/deployments/orgHooks";
import { changesNotDeployed } from "@/lib/features/deployments/types";
import { DEPLOYMENT_STATE, deploymentState, RUN_ACTION } from "@/lib/features/deployments/runState";
import { lastRunAt } from "@/lib/features/projects/deploymentSummary";
import { timeAgo } from "@/lib/features/projects/timeAgo";
import { ChevronRight } from "lucide-react";
import Link from "next/link";
import { useMemo, useState } from "react";
import NewDiagramButton from "../_components/NewDiagramDialog";
import { useProjectContext } from "../_contexts/ProjectContext";

type Row = NonNullable<ReturnType<typeof useOrgDeployments>["data"]>[number];

// Every deployment of every diagram in the project: where it builds, how it
// stands and when it last ran. Each row opens its diagram on the Deploy view
// (?panel=deploy), which acts on it. The list polls while a run is in progress.

// How a run's status reads before "5 minutes ago"; the action word is runState's.
const RUN_WORD: Record<string, string> = { queued: "asked", running: "started", completed: "finished", failed: "failed" };

function CloudsCell({ row }: { row: Row }) {
  // A split deployment runs some blocks on other connections of the team.
  const parts = [
    { connector: row.connector, name: row.connection_name, region: row.region },
    ...(row.placements ?? []).map((p) => ({ connector: p.connector, name: p.connection_name, region: p.region })),
  ];
  return (
    <div className="min-w-0 space-y-1">
      {parts.map((part, i) => (
        <div key={`${part.connector}-${part.name}-${i}`} className="flex min-w-0 items-center gap-2">
          <CloudLogo connector={part.connector} size={18} />
          <span className="text-foreground truncate">{part.name}</span>
          {part.region && <span className="text-muted-foreground truncate text-xs md:hidden">{part.region}</span>}
        </div>
      ))}
    </div>
  );
}

function RegionCell({ row }: { row: Row }) {
  const regions = [...new Set([row.region, ...(row.placements ?? []).map((p) => p.region)].filter(Boolean))];
  return <span className="text-muted-foreground">{regions.join(", ") || "Default"}</span>;
}

function DeploymentRow({ row, diagramName, href, now }: { row: Row; diagramName: string; href: string; now: Date }) {
  const state = DEPLOYMENT_STATE[deploymentState(row)];
  const at = lastRunAt(row);
  const run = row.latest_run;
  const when = timeAgo(at, now);
  const behind = changesNotDeployed(row);

  return (
    <li
      data-testid="project-deployment-row"
      className="group hover:bg-accent/40 focus-within:bg-accent/40 relative grid gap-3 px-4 py-4 transition-colors md:grid-cols-[minmax(0,1.3fr)_minmax(0,1.3fr)_minmax(0,0.8fr)_minmax(0,1fr)_minmax(0,1fr)_1.5rem] md:items-center md:gap-4 md:px-5">
      <div className="min-w-0">
        {/* The diagram's name covers the row, so the whole row opens it. */}
        <Link href={href} className="text-foreground block truncate rounded-sm font-medium outline-none after:absolute after:inset-0 focus-visible:underline">
          {diagramName}
        </Link>
        {behind && (
          <Badge variant="warning" className="mt-1.5">
            Changes not deployed yet
          </Badge>
        )}
      </div>
      <div className="min-w-0 text-sm">
        <span className="text-muted-foreground mb-1 block text-xs md:hidden">Cloud</span>
        <CloudsCell row={row} />
      </div>
      <div className="hidden min-w-0 truncate text-sm md:block">
        <RegionCell row={row} />
      </div>
      <div className="text-sm" data-testid="project-deployment-status" data-status={state.label}>
        <StatusDot tone={state.tone} pulse={state.running} label={state.label} />
      </div>
      <div className="text-muted-foreground text-sm">
        {run && when ? (
          <>
            <span className="text-foreground">{RUN_ACTION[run.action] ?? run.action}</span> {RUN_WORD[run.status] ?? run.status} {when}
          </>
        ) : (
          "Never run"
        )}
      </div>
      <ChevronRight className="text-muted-foreground group-hover:text-link hidden size-4 transition-colors md:block" aria-hidden="true" />
    </li>
  );
}

export default function ProjectDeploymentsPage() {
  const { token, organizationId, projectId, project } = useProjectContext();
  const deploymentsQ = useOrgDeployments(token, organizationId, projectId);
  // Read once per render; "5 minutes ago" needs no ticking, and the list
  // re-renders as it polls.
  const [now] = useState(() => new Date());
  const names = useMemo(() => new Map((project?.diagrams ?? []).map((d) => [d.id, d.name])), [project?.diagrams]);

  const rows = useMemo(
    () =>
      [...(deploymentsQ.data ?? [])].sort(
        (a, b) =>
          (names.get(a.diagram_id) ?? "").localeCompare(names.get(b.diagram_id) ?? "", undefined, { sensitivity: "base" }) ||
          a.created_at.localeCompare(b.created_at),
      ),
    [deploymentsQ.data, names],
  );

  if (!project) return null;
  const diagrams = project.diagrams ?? [];

  if (deploymentsQ.isLoading) {
    return (
      <div role="status" className="surface-card overflow-hidden">
        <span className="sr-only">Loading deployments…</span>
        {[0, 1, 2].map((i) => (
          <div key={i} className="flex items-center gap-4 border-b px-5 py-4 last:border-b-0">
            <Skeleton className="h-5 w-40" />
            <Skeleton className="h-5 w-32" />
            <Skeleton className="ml-auto h-5 w-20" />
          </div>
        ))}
      </div>
    );
  }

  if (deploymentsQ.isError) {
    return (
      <ErrorState
        title="Deployments could not load"
        description="Check your connection, then try again."
        onRetry={() => void deploymentsQ.refetch()}
        retrying={deploymentsQ.isFetching}
      />
    );
  }

  if (rows.length === 0) {
    return (
      <EmptyState
        illustration={<IsoStack preset="tower" accent="volume" size={24} assemble idle="lift" />}
        title="Nothing deployed yet"
        description={
          diagrams.length > 0
            ? "Open a diagram and add a deployment in its Deploy view. Each one shows here with its cloud and status."
            : "Make a diagram first, then deploy it from its Deploy view. Each deployment shows here."
        }
        action={
          diagrams.length > 0 ? (
            <Button asChild>
              <Link href={`/projects/${projectId}/diagrams`}>Go to diagrams</Link>
            </Button>
          ) : (
            <NewDiagramButton />
          )
        }
      />
    );
  }

  return (
    <section aria-label="Deployments" className="surface-card overflow-hidden">
      <div className="text-muted-foreground hidden grid-cols-[minmax(0,1.3fr)_minmax(0,1.3fr)_minmax(0,0.8fr)_minmax(0,1fr)_minmax(0,1fr)_1.5rem] gap-4 border-b px-5 py-2.5 text-xs font-medium md:grid" aria-hidden="true">
        <span>Diagram</span>
        <span>Cloud</span>
        <span>Region</span>
        <span>Status</span>
        <span>Last run</span>
        <span />
      </div>
      <ul className="divide-y">
        {rows.map((row) => (
          <DeploymentRow
            key={row.id}
            row={row}
            diagramName={names.get(row.diagram_id) ?? "Diagram"}
            href={`/projects/${projectId}/diagram/${row.diagram_id}?panel=deploy`}
            now={now}
          />
        ))}
      </ul>
    </section>
  );
}
