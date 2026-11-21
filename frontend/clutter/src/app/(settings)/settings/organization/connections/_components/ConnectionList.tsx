"use client";

import CloudMark from "@/components/clouds/CloudMark";
import ConfirmDialog from "@/components/common/ConfirmDialog";
import StatusDot from "@/components/common/StatusDot";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { toast } from "@/components/ui/sonner";
import { adapterErrorMessage } from "@/lib/adapterApi";
import { cloudLookFor, type CloudLook } from "@/lib/features/catalog/display";
import type { CatalogEntry } from "@/lib/features/catalog/types";
import { connectionStatusLabel } from "@/lib/features/connections/display";
import { deleteErrorMessage, deleteWarning, holdingDiagrams, type Holder } from "@/lib/features/connections/deleteConnection";
import { useDeleteConnection, useVerifyConnection } from "@/lib/features/connections/hooks";
import type { Connection } from "@/lib/features/connections/types";
import { useOrgDeployments } from "@/lib/features/deployments/orgHooks";
import { useProjects } from "@/lib/features/projects/hooks";
import { cn } from "@/lib/utils";
import { Loader2, RefreshCw, Trash2 } from "lucide-react";
import Link from "next/link";
import { useId, useState } from "react";

type Props = {
  token: string;
  organizationId: string;
  connections: Connection[];
  catalog: CatalogEntry[];
  // The connection made a moment ago, marked so the viewer sees where it went.
  justConnectedId: string | null;
};

const checkedAt = (iso: string) => new Date(iso).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });

export default function ConnectionList({ token, organizationId, connections, catalog, justConnectedId }: Props) {
  const ordered = [...connections].sort((a, b) => a.name.localeCompare(b.name) || a.id.localeCompare(b.id));
  // The team's deployments and diagrams, to turn Delete off up front for a
  // connection a deployment may still hold resources on, as the project and
  // team deletes do. While they load, or if they fail, Delete stays on and B
  // refuses the delete with its own reason.
  const deploymentsQ = useOrgDeployments(token, organizationId);
  const projectsQ = useProjects(token, organizationId);
  return (
    <ul className="space-y-3" aria-label="Connections">
      {ordered.map((c) => (
        <ConnectionRow
          key={c.id}
          token={token}
          organizationId={organizationId}
          connection={c}
          cloud={cloudLookFor(catalog, c.connector)}
          justConnected={c.id === justConnectedId}
          holders={holdingDiagrams(c.id, deploymentsQ.data ?? [], projectsQ.data ?? [])}
        />
      ))}
    </ul>
  );
}

function ConnectionRow({
  token,
  organizationId,
  connection,
  cloud,
  justConnected,
  holders,
}: {
  token: string;
  organizationId: string;
  connection: Connection;
  cloud: CloudLook;
  justConnected: boolean;
  holders: Holder[];
}) {
  const held = holders.length > 0;
  const heldId = useId();
  const verifyM = useVerifyConnection(token);
  const deleteM = useDeleteConnection(token);
  const [confirming, setConfirming] = useState(false);
  const working = connection.status === "working";
  const busy = verifyM.isPending || deleteM.isPending;

  const verify = () =>
    verifyM.mutate(
      { organizationId, connectionId: connection.id },
      {
        onSuccess: (c) => {
          if (c.status === "working") toast.success(`${c.name} works`, { description: "Clutter checked it just now." });
          else toast.error(`${c.name} is not working`, { description: c.status_detail ?? undefined });
        },
      },
    );

  return (
    <li
      data-testid="connection-row"
      className={cn(
        "surface-card px-4 py-4 transition-shadow motion-base sm:px-5",
        justConnected && "ring-success/30 ring-1 animate-fade-up [--card-border:color-mix(in_oklch,var(--success)_40%,transparent)]",
      )}>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex min-w-0 items-start gap-3.5">
          <span className="bg-background flex size-10 shrink-0 items-center justify-center rounded-lg border">
            <CloudMark cloud={cloud} size={24} />
          </span>
          <div className="min-w-0 space-y-1">
            <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
              <span className="text-foreground min-w-0 truncate font-medium" title={connection.name}>
                {connection.name}
              </span>
              {justConnected && <Badge variant="success">Just connected</Badge>}
            </div>
            <p className="text-muted-foreground flex flex-wrap items-center gap-x-1.5 text-xs">
              <span>{cloud.title}</span>
              <span aria-hidden="true">·</span>
              <span className="font-mono">{connection.region}</span>
            </p>
            <div className="flex flex-wrap items-center gap-x-1.5 text-xs">
              <StatusDot tone={working ? "success" : "danger"} label={connectionStatusLabel(connection.status)} className={cn("text-xs", working ? "text-success" : "text-danger")} />
              {connection.verified_at && (
                <>
                  <span aria-hidden="true" className="text-muted-foreground">
                    ·
                  </span>
                  <span className="text-muted-foreground">checked {checkedAt(connection.verified_at)}</span>
                </>
              )}
            </div>
            {!working && connection.status_detail && <p className="text-danger max-w-prose text-xs">{connection.status_detail}</p>}
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-2 sm:self-center">
          <Button type="button" size="sm" variant="outline" disabled={busy} onClick={verify}>
            {verifyM.isPending ? <Loader2 className="animate-spin" aria-hidden /> : <RefreshCw aria-hidden />}
            {verifyM.isPending ? "Checking…" : "Verify"}
          </Button>
          <Button
            type="button"
            size="sm"
            variant="ghost"
            disabled={busy || held}
            aria-describedby={held ? heldId : undefined}
            onClick={() => {
              deleteM.reset();
              setConfirming(true);
            }}
            className="text-muted-foreground hover:text-danger"
            aria-label={`Delete ${connection.name}`}>
            <Trash2 aria-hidden />
            Delete
          </Button>
        </div>
      </div>
      {verifyM.isError && (
        <p role="alert" className="text-danger mt-3 text-xs">
          Could not check this connection. {adapterErrorMessage(verifyM.error)}
        </p>
      )}
      {held && (
        <p id={heldId} data-testid="delete-connection-held" className="text-foreground mt-3 text-xs">
          {holders.length === 1 ? "A deployment of " : "Deployments of "}
          {holders.map((h, i) => (
            <span key={h.diagramId}>
              {i > 0 && (i === holders.length - 1 ? " and " : ", ")}
              <Link href={h.href} className="text-link font-medium underline-offset-4 hover:underline">
                {h.name}
              </Link>
            </span>
          ))}{" "}
          may still hold resources on this connection, so Delete is off. Destroy {holders.length === 1 ? "it" : "them"} first.
        </p>
      )}

      {confirming && (
        <ConfirmDialog
          title={`Delete ${connection.name}?`}
          confirmLabel="Delete connection"
          pendingLabel="Deleting…"
          isPending={deleteM.isPending}
          error={deleteM.isError ? deleteErrorMessage(deleteM.error) : null}
          onCancel={() => setConfirming(false)}
          onConfirm={() =>
            deleteM.mutate(
              { organizationId, connectionId: connection.id },
              {
                onSuccess: () => {
                  setConfirming(false);
                  toast.success(`Deleted ${connection.name}`);
                },
              },
            )
          }>
          <p data-testid="delete-connection-warning">{deleteWarning(connection.diagrams ?? 0)}</p>
        </ConfirmDialog>
      )}
    </li>
  );
}
