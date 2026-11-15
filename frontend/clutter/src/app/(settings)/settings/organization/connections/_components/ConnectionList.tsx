"use client";

import CloudMark from "@/components/clouds/CloudMark";
import { Button } from "@/components/ui/button";
import { adapterErrorMessage } from "@/lib/adapterApi";
import { cloudLookFor, cloudTitleWithNote, type CloudLook } from "@/lib/features/catalog/display";
import type { CatalogEntry } from "@/lib/features/catalog/types";
import { deleteErrorMessage, deleteWarning } from "@/lib/features/connections/deleteConnection";
import { useDeleteConnection, useVerifyConnection } from "@/lib/features/connections/hooks";
import type { Connection } from "@/lib/features/connections/types";
import { CheckCircle2, Loader2, RefreshCw, Trash2, XCircle } from "lucide-react";
import { useState } from "react";
import { Notice } from "./Notice";

type Props = {
  token: string;
  organizationId: string;
  connections: Connection[];
  catalog: CatalogEntry[];
};

export default function ConnectionList({ token, organizationId, connections, catalog }: Props) {
  if (connections.length === 0) {
    return <Notice tone="info">No connections yet. Add one below to deploy building-block diagrams.</Notice>;
  }
  return (
    <ul className="space-y-3">
      {connections.map((c) => (
        <ConnectionRow key={c.id} token={token} organizationId={organizationId} connection={c} cloud={cloudLookFor(catalog, c.connector)} />
      ))}
    </ul>
  );
}

function ConnectionRow({ token, organizationId, connection, cloud }: { token: string; organizationId: string; connection: Connection; cloud: CloudLook }) {
  const verifyM = useVerifyConnection(token);
  const deleteM = useDeleteConnection(token);
  const [confirming, setConfirming] = useState(false);
  const working = connection.status === "working";
  const busy = verifyM.isPending || deleteM.isPending;

  return (
    <li data-testid="connection-row" className="rounded-xl border border-slate-800 bg-slate-900/50 px-4 py-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex min-w-0 items-start gap-3">
          <CloudMark cloud={cloud} size={28} className="mt-0.5" />
          <div className="min-w-0">
            <div className="flex min-w-0 items-center gap-2">
              <span className="min-w-0 truncate font-medium text-white" title={connection.name}>
                {connection.name}
              </span>
              <span className="shrink-0 whitespace-nowrap rounded border border-slate-700 px-1.5 py-0.5 text-xs text-slate-300">{cloudTitleWithNote(cloud)}</span>
              <span className="shrink-0 whitespace-nowrap text-xs text-slate-500">{connection.region}</span>
            </div>
            <div className={`mt-1 flex items-center gap-1.5 text-xs ${working ? "text-emerald-300" : "text-red-300"}`}>
              {working ? <CheckCircle2 className="h-3.5 w-3.5" /> : <XCircle className="h-3.5 w-3.5" />}
              <span>
                {working ? "Verified" : "Not working"}
                {connection.verified_at && <span className="text-slate-500">, checked {new Date(connection.verified_at).toLocaleString()}</span>}
              </span>
            </div>
            {!working && connection.status_detail && <p className="mt-1 text-xs text-red-300">{connection.status_detail}</p>}
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button
            type="button"
            size="sm"
            variant="outline"
            disabled={busy}
            onClick={() => verifyM.mutate({ organizationId, connectionId: connection.id })}
            className="border-slate-700 bg-slate-800 text-slate-200 hover:bg-slate-700">
            {verifyM.isPending ? <Loader2 className="animate-spin" /> : <RefreshCw />}
            Verify
          </Button>
          {confirming ? (
            <>
              <Button
                type="button"
                size="sm"
                disabled={busy}
                onClick={() => deleteM.mutate({ organizationId, connectionId: connection.id }, { onSettled: () => setConfirming(false) })}
                className="bg-red-600 text-white hover:bg-red-500">
                {deleteM.isPending ? <Loader2 className="animate-spin" /> : <Trash2 />}
                Delete for good
              </Button>
              <Button type="button" size="sm" variant="ghost" disabled={busy} onClick={() => setConfirming(false)} className="text-slate-300">
                Cancel
              </Button>
            </>
          ) : (
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={busy}
              onClick={() => {
                deleteM.reset();
                setConfirming(true);
              }}
              className="border-red-800 bg-transparent text-red-300 hover:bg-red-950 hover:text-red-200">
              <Trash2 />
              Delete
            </Button>
          )}
        </div>
      </div>
      {confirming && (
        <p role="alert" data-testid="delete-connection-warning" className="mt-2 text-xs text-amber-200">
          {deleteWarning(connection.diagrams ?? 0)}
        </p>
      )}
      {verifyM.isError && <p className="mt-2 text-xs text-red-300">Could not check this connection: {adapterErrorMessage(verifyM.error)}</p>}
      {deleteM.isError && (
        <p role="alert" className="mt-2 text-sm text-red-300">
          {deleteErrorMessage(deleteM.error)}
        </p>
      )}
    </li>
  );
}
