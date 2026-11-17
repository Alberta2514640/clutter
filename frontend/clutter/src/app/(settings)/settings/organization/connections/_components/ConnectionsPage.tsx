"use client";

import { adapterApiConfigured, adapterErrorMessage, NOT_CONFIGURED_MESSAGE } from "@/lib/adapterApi";
import { useCatalog } from "@/lib/features/catalog/hooks";
import { useConnections } from "@/lib/features/connections/hooks";
import { useCurrentOrganization } from "@/lib/features/organization/hooks";
import { useMe } from "@/lib/features/user/hooks";
import { apiErrorMessage } from "@/lib/features/user/session";
import AddConnection from "./AddConnection";
import ConnectionList from "./ConnectionList";
import { Loading, Notice } from "./Notice";

export default function ConnectionsPage() {
  const meQ = useMe();
  const token = meQ.data?.token ?? null;
  const currentOrg = useCurrentOrganization(token);
  const orgId = currentOrg.organizationId;
  const catalogQ = useCatalog(token);
  const connectionsQ = useConnections(token, orgId);

  if (!adapterApiConfigured) {
    return <Notice tone="error">{NOT_CONFIGURED_MESSAGE}</Notice>;
  }
  if (meQ.isLoading || currentOrg.isLoading) {
    return <Loading text="Loading your organization…" />;
  }
  if (!token) return <Notice tone="error">Sign in to see your connections.</Notice>;
  // The organizations come from A's API, whose error body is not for people.
  if (currentOrg.isError && !currentOrg.data) {
    const reason = apiErrorMessage(currentOrg.error);
    return <Notice tone="error">{reason ? `Could not load your organizations: ${reason}` : "Could not load your organizations."}</Notice>;
  }
  if (!currentOrg.organization) return <Notice tone="info">Create an organization first. Connections belong to an organization.</Notice>;

  const catalog = catalogQ.data ?? [];

  return (
    <div className="space-y-8 pb-12 md:px-6">
      <div>
        <h2 className="text-xl font-semibold text-white">Connections</h2>
        <p className="mt-1 text-sm text-slate-400">
          The cloud accounts <span className="text-slate-200">{currentOrg.organization.name}</span> deploys to. Every diagram and
          teammate in this organization can use them. Clutter checks each one before it saves it.
        </p>
      </div>

      <section aria-labelledby="connections-list-title" className="space-y-3">
        <h3 id="connections-list-title" className="text-sm font-semibold uppercase tracking-wide text-slate-400">
          Your connections
        </h3>
        {connectionsQ.isLoading ? (
          <Loading text="Loading connections…" />
        ) : connectionsQ.isError && !connectionsQ.data ? (
          <Notice tone="error">Could not load connections: {adapterErrorMessage(connectionsQ.error)}</Notice>
        ) : (
          <ConnectionList token={token} organizationId={currentOrg.organization.id} connections={connectionsQ.data ?? []} catalog={catalog} />
        )}
      </section>

      <section aria-labelledby="connections-add-title" className="space-y-3">
        <h3 id="connections-add-title" className="text-sm font-semibold uppercase tracking-wide text-slate-400">
          Add a connection
        </h3>
        {catalogQ.isLoading ? (
          <Loading text="Loading the clouds Clutter offers…" />
        ) : catalogQ.isError && !catalogQ.data ? (
          <Notice tone="error">Could not load the clouds Clutter offers: {adapterErrorMessage(catalogQ.error)}</Notice>
        ) : catalog.length === 0 ? (
          <Notice tone="info">Clutter offers no clouds yet.</Notice>
        ) : (
          <AddConnection token={token} organizationId={currentOrg.organization.id} catalog={catalog} />
        )}
      </section>
    </div>
  );
}
