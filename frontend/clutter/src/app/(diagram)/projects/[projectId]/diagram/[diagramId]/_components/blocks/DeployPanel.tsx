"use client";

import { adapterApiConfigured, adapterErrorMessage, NOT_CONFIGURED_MESSAGE } from "@/lib/adapterApi";
import type { CatalogEntry } from "@/lib/features/catalog/types";
import { useConnections } from "@/lib/features/connections/hooks";
import type { BlockRef } from "@/lib/features/deployments/compatibility";
import { useDiagramDeployments } from "@/lib/features/deployments/hooks";
import { Cloud, Loader2 } from "lucide-react";
import AddDeployment from "./AddDeployment";
import ConnectionsLink from "./ConnectionsLink";
import DeploymentRow from "./DeploymentRow";

type Props = {
  token: string | null;
  organizationId: string | null;
  diagramId: string;
  catalog: CatalogEntry[];
  // The blocks as last saved: B checks and runs the saved diagram.
  savedBlocks: BlockRef[];
  // The blocks on the canvas now, which a new deployment is for.
  currentBlocks: BlockRef[];
  // The saved servers the diagram makes a website (results.servesWeb).
  webServers: ReadonlySet<string>;
  blockTitle: (kind: string) => string;
  dirty: boolean;
  // Saves the canvas; false when it could not.
  save: () => Promise<boolean>;
};

// The deploy panel: one row per deployment of this diagram (one per linked
// cloud), each with its compatibility, settings, preview, runs and log.
export default function DeployPanel({ token, organizationId, diagramId, catalog, savedBlocks, currentBlocks, webServers, blockTitle, dirty, save }: Props) {
  const deploymentsQ = useDiagramDeployments(token, organizationId, diagramId);
  const connectionsQ = useConnections(token, organizationId);

  if (!adapterApiConfigured) {
    return <p className="text-sm text-red-300">{NOT_CONFIGURED_MESSAGE}</p>;
  }
  if (!organizationId || deploymentsQ.isLoading) {
    return (
      <div role="status" className="flex items-center gap-2 text-sm text-slate-400">
        <Loader2 className="h-4 w-4 animate-spin" />
        Loading deployments…
      </div>
    );
  }
  if (deploymentsQ.isError) {
    return (
      <p role="alert" className="text-sm text-red-300">
        Could not load deployments: {adapterErrorMessage(deploymentsQ.error)}
      </p>
    );
  }

  const deployments = deploymentsQ.data ?? [];
  const connections = connectionsQ.data ?? [];
  const used = new Set(deployments.map((d) => d.connection_id));
  const blockedReason = dirty ? "Save the diagram first. Deployments use the saved version." : null;

  // No connection at all: the one thing to do is connect a cloud.
  if (connectionsQ.isSuccess && connections.length === 0 && deployments.length === 0) {
    return (
      <div data-testid="connect-a-cloud" className="space-y-3 rounded-lg border border-slate-700 bg-slate-900/60 p-4">
        <h3 className="text-sm font-semibold text-white">Connect a cloud to deploy</h3>
        <p className="text-sm text-slate-300">Clutter builds this diagram in your own cloud account, so it needs a connection to it first.</p>
        <ConnectionsLink
          organizationId={organizationId}
          className="inline-flex items-center gap-1.5 rounded-md bg-teal-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-teal-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-teal-300">
          <Cloud className="h-4 w-4" aria-hidden />
          Connect a cloud
        </ConnectionsLink>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {deployments.length === 0 ? (
        <p className="text-sm text-slate-400">
          {currentBlocks.length === 0
            ? "Add building blocks. Then add a deployment for each cloud it should run in."
            : "No deployments yet. Add one for each cloud this diagram should run in."}
        </p>
      ) : (
        <ul className="space-y-3">
          {deployments.map((d) => (
            <DeploymentRow
              key={d.id}
              token={token}
              organizationId={organizationId}
              diagramId={diagramId}
              item={d}
              entry={catalog.find((e) => e.manifest.connector === d.connector) ?? null}
              catalog={catalog}
              connections={connections}
              blocks={savedBlocks}
              blockTitle={blockTitle}
              webServers={webServers}
              blockedReason={blockedReason}
            />
          ))}
        </ul>
      )}
      {connectionsQ.isLoading ? (
        <div role="status" className="flex items-center gap-2 text-xs text-slate-400">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          Loading connections…
        </div>
      ) : connectionsQ.isError ? (
        <p className="text-xs text-red-300">Could not load connections: {adapterErrorMessage(connectionsQ.error)}</p>
      ) : (
        <AddDeployment
          token={token}
          organizationId={organizationId}
          diagramId={diagramId}
          connections={connections}
          usedConnectionIds={used}
          catalog={catalog}
          blocks={currentBlocks}
          blockTitle={blockTitle}
          saveFirst={dirty ? save : null}
        />
      )}
    </div>
  );
}
