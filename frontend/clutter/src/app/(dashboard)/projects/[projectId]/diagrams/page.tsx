"use client";

import IsoStack from "@/components/brand/IsoStack";
import CloudLogo, { CLOUD_LOGOS } from "@/components/common/CloudLogo";
import ConfirmDialog from "@/components/common/ConfirmDialog";
import EmptyState from "@/components/common/EmptyState";
import StatusDot from "@/components/common/StatusDot";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { toast } from "@/components/ui/sonner";
import { isBlocksDocument } from "@/lib/features/diagram/blocksDocument";
import { editedLabel } from "@/lib/features/diagram/edited";
import { useDeleteDiagram, useDiagrams } from "@/lib/features/diagram/hooks";
import { useOrgDeployments } from "@/lib/features/deployments/orgHooks";
import { DEPLOYMENT_STATE, deploymentState } from "@/lib/features/deployments/runState";
import { holdsResources } from "@/lib/features/projects/deploymentSummary";
import { isConflict } from "@/lib/features/user/session";
import { cn } from "@/lib/utils";
import { ExternalLink, MoreHorizontal, PenTool, Trash2 } from "lucide-react";
import Link from "next/link";
import { useMemo, useState } from "react";
import NewDiagramButton from "../_components/NewDiagramDialog";
import { useProjectContext } from "../_contexts/ProjectContext";

type DeleteTarget = { id: string; name: string } | null;
type Row = NonNullable<ReturnType<typeof useOrgDeployments>["data"]>[number];

// What a diagram's deployments say on its card: the clouds it deploys to,
// and its one most telling state (a run in progress, else live, else failed).
function DeployLine({ rows }: { rows: Row[] }) {
  if (rows.length === 0) return <p className="text-muted-foreground text-xs">Not deployed yet</p>;
  const states = rows.map(deploymentState);
  const key =
    states.find((s) => DEPLOYMENT_STATE[s].running) ?? states.find((s) => s === "live") ?? states.find((s) => s === "failed") ?? states[0];
  const shown = DEPLOYMENT_STATE[key];
  const live = states.filter((s) => s === "live").length;
  const connectors = [...new Set(rows.flatMap((r) => [r.connector, ...(r.placements ?? []).map((p) => p.connector)]))];
  const names = connectors.map((c) => CLOUD_LOGOS[c]?.shortName ?? c);
  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5">
      <span className="flex items-center gap-1" aria-label={`Deploys to ${names.join(", ")}`} role="img">
        {connectors.map((c) => (
          <CloudLogo key={c} connector={c} size={16} />
        ))}
      </span>
      <StatusDot
        tone={shown.tone}
        pulse={shown.running}
        label={key === "live" && live > 1 ? `${live} live` : shown.label}
        className="text-muted-foreground text-xs"
      />
    </div>
  );
}

export default function ProjectDiagramsPage() {
  const { projectId, token, project, organizationId } = useProjectContext();

  // The project lists only ids and names; the diagrams themselves say which
  // editor each one uses, for the older-editor mark.
  const diagramsQ = useDiagrams(token, projectId);
  const loadedById = useMemo(() => new Map((diagramsQ.data ?? []).map((d) => [d.id, d])), [diagramsQ.data]);
  const deploymentsQ = useOrgDeployments(token, organizationId, projectId);
  const rowsByDiagram = useMemo(() => {
    const out = new Map<string, Row[]>();
    for (const row of deploymentsQ.data ?? []) out.set(row.diagram_id, [...(out.get(row.diagram_id) ?? []), row]);
    return out;
  }, [deploymentsQ.data]);
  // Read once per render; the cards' "Edited 5 minutes ago" needs no ticking.
  const [now] = useState(() => new Date());

  const deleteDiagramM = useDeleteDiagram(token);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<DeleteTarget>(null);

  if (!project) return null;

  const handleDeleteConfirm = async () => {
    if (!deleteTarget) return;
    setDeleteError(null);
    try {
      await deleteDiagramM.mutateAsync({ organizationId: project.organizationId, projectId, diagramId: deleteTarget.id });
    } catch (err) {
      setDeleteError(
        isConflict(err)
          ? "This diagram still has deployments that may hold resources. Destroy them in its Deploy view, then delete it."
          : "Clutter could not delete the diagram. Try again.",
      );
      return;
    }
    toast.success(`Deleted "${deleteTarget.name}"`);
    setDeleteTarget(null);
  };

  const diagrams = project.diagrams ?? [];

  if (diagrams.length === 0) {
    return (
      <EmptyState
        illustration={<IsoStack preset="linked" accent="vm" size={24} assemble idle="lift" />}
        title="No diagrams yet"
        description="Draw what you want to build in plain building blocks, then deploy it to a connected cloud. Start blank or from an example."
        action={<NewDiagramButton />}
      />
    );
  }

  return (
    <>
      {deleteTarget && (
        <ConfirmDialog
          title="Delete this diagram?"
          confirmLabel="Delete diagram"
          pendingLabel="Deleting…"
          isPending={deleteDiagramM.isPending}
          error={deleteError}
          onCancel={() => setDeleteTarget(null)}
          onConfirm={() => void handleDeleteConfirm()}>
          <p>
            <span className="text-foreground font-medium">{deleteTarget.name}</span> is deleted for good. This cannot be undone.
          </p>
          <p>Clutter refuses while one of its deployments may still hold resources in your cloud.</p>
        </ConfirmDialog>
      )}

      <ul className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {diagrams.map((diagram) => {
          const isDeleting = deleteDiagramM.isPending && deleteDiagramM.variables?.diagramId === diagram.id;
          const loaded = loadedById.get(diagram.id);
          // Unknown until the diagrams load; only a loaded, unmarked diagram is A's.
          const isOlderAwsDiagram = !!loaded && !isBlocksDocument(loaded.data);
          const edited = editedLabel(loaded?.latestUpdateAt ?? loaded?.createdAt, now);
          const rows = rowsByDiagram.get(diagram.id) ?? [];
          const holding = rows.some((r) => holdsResources(r.status));
          const href = `/projects/${projectId}/diagram/${diagram.id}`;

          return (
            <li key={diagram.id} className="animate-fade-in">
              <article
                data-testid="diagram-card"
                className={cn(
                  "group surface-card surface-card-interactive text-card-foreground relative flex h-full flex-col p-5",
                  isDeleting && "opacity-50",
                )}>
                <div className="flex items-start gap-3">
                  <span className="bg-muted text-muted-foreground group-hover:bg-primary/10 group-hover:text-link flex size-10 shrink-0 items-center justify-center rounded-lg transition-colors">
                    <PenTool className="size-5" aria-hidden="true" />
                  </span>
                  <div className="min-w-0 flex-1 space-y-0.5">
                    {/* The title link covers the card, so the whole card opens the diagram. */}
                    <Link
                      href={href}
                      className="block truncate rounded-sm font-semibold outline-none after:absolute after:inset-0 after:rounded-xl focus-visible:underline">
                      {diagram.name}
                    </Link>
                    {edited && <p className="text-muted-foreground text-xs">{edited}</p>}
                  </div>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button variant="ghost" size="icon-sm" aria-label={`Actions for ${diagram.name}`} disabled={isDeleting} className="relative z-10 -mt-1 -mr-2">
                        <MoreHorizontal aria-hidden="true" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end" className="min-w-44">
                      <DropdownMenuItem asChild>
                        <Link href={href}>
                          <ExternalLink aria-hidden="true" />
                          Open
                        </Link>
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem
                        variant="destructive"
                        onSelect={() => {
                          setDeleteError(null);
                          setDeleteTarget({ id: diagram.id, name: diagram.name });
                        }}>
                        <Trash2 aria-hidden="true" />
                        Delete
                        {holding && <span className="sr-only"> (it may still hold resources, so Clutter will refuse)</span>}
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>

                <div className="mt-auto space-y-2 pt-4">
                  {isOlderAwsDiagram && <Badge variant="warning">AWS only, older editor</Badge>}
                  {!isOlderAwsDiagram && (deploymentsQ.isSuccess ? <DeployLine rows={rows} /> : <div className="h-4" />)}
                </div>
              </article>
            </li>
          );
        })}
      </ul>
    </>
  );
}
