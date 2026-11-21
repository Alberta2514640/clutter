"use client";

import CloudLogo from "@/components/common/CloudLogo";
import ErrorState from "@/components/common/ErrorState";
import StatusDot from "@/components/common/StatusDot";
import ConnectCloudSheet from "@/components/connections/ConnectCloudSheet";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { connectionStatusLabel } from "@/lib/features/connections/display";
import { useConnections } from "@/lib/features/connections/hooks";
import { ArrowRight, Cable } from "lucide-react";
import Link from "next/link";
import { useState } from "react";

const CONNECTIONS_HREF = "/settings/organization/connections";
const SHOWN = 4;

// The team's connections at a glance: each one's cloud, name and whether it
// works, with the way to Connections.
export default function ConnectionsSummary({ token, organizationId }: { token: string; organizationId: string }) {
  const connectionsQ = useConnections(token, organizationId);
  const connections = connectionsQ.data ?? [];
  // Connecting happens in a side sheet, without leaving Projects.
  const [connecting, setConnecting] = useState(false);
  const failed = connections.filter((c) => c.status === "failed").length;

  return (
    <section aria-labelledby="connections-heading" className="surface-card p-5">
      <div className="flex items-center gap-2">
        <Cable className="text-muted-foreground size-4" aria-hidden="true" />
        <h2 id="connections-heading" className="text-foreground text-sm font-semibold">
          Connections
        </h2>
      </div>
      <p className="text-muted-foreground mt-1 text-sm">Cloud accounts this team can build in.</p>

      <div className="mt-4">
        {connectionsQ.isLoading ? (
          <div role="status" className="space-y-3">
            <span className="sr-only">Loading connections…</span>
            <Skeleton className="h-5 w-full" />
            <Skeleton className="h-5 w-3/4" />
          </div>
        ) : connectionsQ.isError ? (
          <ErrorState
            variant="inline"
            description="Connections could not load."
            onRetry={() => void connectionsQ.refetch()}
            retrying={connectionsQ.isFetching}
          />
        ) : connections.length === 0 ? (
          <div className="space-y-3">
            <p className="text-foreground text-sm">No cloud connected yet. Connect one to deploy.</p>
            <Button size="sm" variant="outline" onClick={() => setConnecting(true)}>
              Connect a cloud
            </Button>
          </div>
        ) : (
          <>
            <ul className="space-y-3">
              {connections.slice(0, SHOWN).map((c) => (
                <li key={c.id} className="flex min-w-0 items-center gap-2.5 text-sm">
                  <CloudLogo connector={c.connector} size={18} />
                  <span className="text-foreground min-w-0 flex-1 truncate">{c.name}</span>
                  <StatusDot tone={c.status === "working" ? "success" : "danger"} label={connectionStatusLabel(c.status)} className="text-muted-foreground shrink-0 text-xs" />
                </li>
              ))}
            </ul>
            {connections.length > SHOWN && <p className="text-muted-foreground mt-3 text-xs">And {connections.length - SHOWN} more.</p>}
            {failed > 0 && (
              <p className="text-muted-foreground mt-3 text-xs">
                {failed === 1 ? "One connection is not working." : `${failed} connections are not working.`} Check it on Connections.
              </p>
            )}
          </>
        )}
      </div>

      {connections.length > 0 && (
        <Link
          href={CONNECTIONS_HREF}
          className="text-link mt-5 inline-flex min-h-6 items-center gap-1 rounded-sm text-sm font-medium underline-offset-4 outline-none hover:underline focus-visible:ring-[3px] focus-visible:ring-ring/50">
          Manage connections
          <ArrowRight className="size-3.5" aria-hidden="true" />
        </Link>
      )}
      <ConnectCloudSheet open={connecting} onOpenChange={setConnecting} organizationId={organizationId} onConnected={() => setConnecting(false)} />
    </section>
  );
}
