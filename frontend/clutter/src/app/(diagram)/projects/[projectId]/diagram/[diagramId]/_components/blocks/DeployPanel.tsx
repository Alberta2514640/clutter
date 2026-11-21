"use client";

import ConnectCloudSheet from "@/components/connections/ConnectCloudSheet";
import EmptyState from "@/components/common/EmptyState";
import ErrorState from "@/components/common/ErrorState";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { adapterApiConfigured, adapterErrorMessage, NOT_CONFIGURED_MESSAGE } from "@/lib/adapterApi";
import { entryFor } from "@/lib/features/catalog/forms";
import type { CatalogEntry } from "@/lib/features/catalog/types";
import { useConnections } from "@/lib/features/connections/hooks";
import type { BlockRef } from "@/lib/features/deployments/compatibility";
import { useDiagramDeployments } from "@/lib/features/deployments/hooks";
import { usePreferencesStore } from "@/lib/features/preferences/uiStore";
import { Cloud } from "lucide-react";
import { useState } from "react";
import AddDeployment from "./AddDeployment";
import DeploymentRow from "./DeploymentRow";

type Props = {
  token: string | null;
  organizationId: string | null;
  // Why the diagram's organization cannot be found, or null while it is found
  // or still being looked up.
  organizationError: string | null;
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
  // The deployment the canvas shows, and how to choose another.
  selectedId: string | null;
  onSelect: (deploymentId: string) => void;
  // Opens the code panel on a deployment.
  onPreview: (deploymentId: string) => void;
};

// The deploy panel: one row per deployment of this diagram (one per linked
// cloud), each with its state, its one next action, compatibility, settings,
// runs and logs. With no connection, the one thing to do is connect a cloud,
// in a side sheet that keeps the editor open.
export default function DeployPanel({
  token,
  organizationId,
  organizationError,
  diagramId,
  catalog,
  savedBlocks,
  currentBlocks,
  webServers,
  blockTitle,
  dirty,
  save,
  selectedId,
  onSelect,
  onPreview,
}: Props) {
  const deploymentsQ = useDiagramDeployments(token, organizationId, diagramId);
  const connectionsQ = useConnections(token, organizationId);
  const firstCloud = usePreferencesStore((s) => s.firstCloud);
  const [connecting, setConnecting] = useState(false);

  if (!adapterApiConfigured) {
    return <ErrorState title="Deploying is not set up" description={NOT_CONFIGURED_MESSAGE} />;
  }
  if (organizationError) {
    return <ErrorState description={organizationError} onRetry={() => window.location.reload()} retryLabel="Reload" />;
  }
  if (!organizationId || deploymentsQ.isLoading) {
    return (
      <div role="status" className="space-y-3">
        <span className="sr-only">Loading deployments…</span>
        <Skeleton className="h-40 rounded-xl" />
        <Skeleton className="h-10 rounded-lg" />
      </div>
    );
  }
  // A failed refetch keeps the rows on screen; only a first load that failed
  // shows the error instead.
  if (deploymentsQ.isError && !deploymentsQ.data) {
    return (
      <ErrorState
        description={`Could not load this diagram's deployments. ${adapterErrorMessage(deploymentsQ.error)}`}
        onRetry={() => void deploymentsQ.refetch()}
        retrying={deploymentsQ.isFetching}
      />
    );
  }

  const deployments = deploymentsQ.data ?? [];
  const connections = connectionsQ.data ?? [];
  const used = new Set(deployments.map((d) => d.connection_id));
  const blockedReason = dirty ? "Save the diagram first. Deployments use the saved version." : null;
  const selected = deployments.find((d) => d.id === selectedId)?.id ?? deployments[0]?.id ?? null;
  const sheet = (
    <ConnectCloudSheet
      open={connecting}
      onOpenChange={setConnecting}
      organizationId={organizationId}
      preferredConnector={firstCloud ?? undefined}
      onConnected={() => setConnecting(false)}
    />
  );

  // No connection at all: the one thing to do is connect a cloud.
  if (connectionsQ.isSuccess && connections.length === 0 && deployments.length === 0) {
    return (
      <div data-testid="connect-a-cloud">
        <EmptyState
          icon={Cloud}
          headingLevel="h3"
          title="Connect a cloud to deploy"
          description="A connection is a cloud account Clutter can build in. Clutter builds this diagram in your own account."
          action={
            <Button type="button" onClick={() => setConnecting(true)}>
              <Cloud aria-hidden />
              Connect a cloud
            </Button>
          }
        />
        {sheet}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {deployments.length === 0 ? (
        <p className="text-muted-foreground text-sm">
          {currentBlocks.length === 0
            ? "Add building blocks. Then add a deployment for each cloud it should run in."
            : "No deployments yet. A deployment builds this diagram on one of your connections. Add one for each cloud it should run in."}
        </p>
      ) : (
        <ul className="space-y-4">
          {deployments.map((d) => (
            <DeploymentRow
              key={d.id}
              token={token}
              organizationId={organizationId}
              diagramId={diagramId}
              item={d}
              entry={entryFor(catalog, d.connector)}
              catalog={catalog}
              connections={connections}
              blocks={savedBlocks}
              blockTitle={blockTitle}
              webServers={webServers}
              blockedReason={blockedReason}
              selected={deployments.length > 1 && d.id === selected}
              defaultOpen={deployments.length === 1 || d.id === selected}
              onSelect={() => {
                if (d.id !== selectedId) onSelect(d.id);
              }}
              onPreview={() => onPreview(d.id)}
            />
          ))}
        </ul>
      )}
      {connectionsQ.isLoading ? (
        <Skeleton className="h-10 rounded-lg" />
      ) : connectionsQ.isError && !connectionsQ.data ? (
        <ErrorState variant="inline" description={`Could not load connections. ${adapterErrorMessage(connectionsQ.error)}`} onRetry={() => void connectionsQ.refetch()} />
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
          onAddConnection={() => setConnecting(true)}
          onAdded={onSelect}
        />
      )}
      {sheet}
    </div>
  );
}
