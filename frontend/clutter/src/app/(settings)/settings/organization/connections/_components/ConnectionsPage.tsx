"use client";

import AssembleInView from "@/components/brand/AssembleInView";
import IsoStack from "@/components/brand/IsoStack";
import EmptyState from "@/components/common/EmptyState";
import ErrorState from "@/components/common/ErrorState";
import PageHeader from "@/components/common/PageHeader";
import ConnectCloudFlow from "@/components/connections/ConnectCloudFlow";
import ConnectCloudSheet from "@/components/connections/ConnectCloudSheet";
import { Notice } from "@/components/connections/Field";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { adapterApiConfigured, adapterErrorMessage, NOT_CONFIGURED_MESSAGE } from "@/lib/adapterApi";
import { useCatalog } from "@/lib/features/catalog/hooks";
import { useConnections } from "@/lib/features/connections/hooks";
import { useCurrentOrganization } from "@/lib/features/organization/hooks";
import { usePreferencesStore } from "@/lib/features/preferences/uiStore";
import { useMe } from "@/lib/features/user/hooks";
import { Plus, Users } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import ConnectionList from "./ConnectionList";

const CONNECTIONS_PATH = "/settings/organization/connections";
const CREATE_TEAM_PATH = "/settings/organization/create-org";

const TITLE = "Connections";

function ListSkeleton() {
  return (
    <div role="status" aria-label="Loading connections" className="space-y-3">
      {[0, 1].map((i) => (
        <Skeleton key={i} className="h-[92px] rounded-xl" />
      ))}
    </div>
  );
}

// The team's connections: the cloud accounts its diagrams deploy to. With no
// connection yet the page is the connection flow itself; after that, "Connect
// a cloud" opens the same flow in a side sheet. ?connect=<connector> (the old
// AWS account page redirects here with aws) opens the flow on that cloud.
export default function ConnectionsPage({ openConnector }: { openConnector: string | null }) {
  const router = useRouter();
  const meQ = useMe();
  const token = meQ.data?.token ?? null;
  const currentOrg = useCurrentOrganization(token);
  const orgId = currentOrg.organizationId;
  const catalogQ = useCatalog(token);
  const connectionsQ = useConnections(token, orgId);
  const firstCloud = usePreferencesStore((s) => s.firstCloud);

  // null: follow ?connect=; true or false once the viewer opened or closed it.
  const [sheetOpen, setSheetOpen] = useState<boolean | null>(null);
  const [sheetCloud, setSheetCloud] = useState<string | null>(openConnector);
  const [justConnectedId, setJustConnectedId] = useState<string | null>(null);

  if (!adapterApiConfigured) {
    return (
      <>
        <PageHeader title={TITLE} />
        <Notice tone="error">{NOT_CONFIGURED_MESSAGE}</Notice>
      </>
    );
  }
  if (meQ.isLoading || currentOrg.isLoading) {
    return (
      <>
        <PageHeader title={TITLE} description="Cloud accounts your team can build in." />
        <ListSkeleton />
      </>
    );
  }
  if (!token) {
    return (
      <>
        <PageHeader title={TITLE} />
        <Notice tone="error">Sign in to see your connections.</Notice>
      </>
    );
  }
  // The organizations come from A's API, whose error body is not for people.
  if (currentOrg.isError && !currentOrg.data) {
    return (
      <>
        <PageHeader title={TITLE} />
        <ErrorState
          title="Could not load your teams"
          description="Clutter could not load your teams. Check your connection and try again."
          onRetry={() => void currentOrg.refetch()}
          retrying={currentOrg.isFetching}
        />
      </>
    );
  }
  const org = currentOrg.organization;
  if (!org) {
    return (
      <>
        <PageHeader title={TITLE} description="Cloud accounts your team can build in." />
        <EmptyState
          icon={Users}
          title="Create a team first"
          description="Connections belong to a team. Create one, then connect a cloud."
          action={
            <Button asChild>
              <Link href={CREATE_TEAM_PATH}>Create a team</Link>
            </Button>
          }
        />
      </>
    );
  }

  const connections = connectionsQ.data ?? [];
  const hasConnections = connections.length > 0;
  const open = sheetOpen ?? (!!openConnector && hasConnections);

  const openSheet = () => {
    setSheetCloud(null);
    setSheetOpen(true);
  };
  const onSheetOpenChange = (next: boolean) => {
    setSheetOpen(next);
    // The flow asked for in the address opens once.
    if (!next && openConnector) router.replace(CONNECTIONS_PATH, { scroll: false });
  };

  return (
    <>
      <PageHeader
        eyebrow={org.name}
        title={TITLE}
        description="Cloud accounts this team can build in. Every diagram in the team can use them, and Clutter checks each one before it saves it."
        actions={
          hasConnections && (
            <Button type="button" onClick={openSheet}>
              <Plus aria-hidden />
              Connect a cloud
            </Button>
          )
        }
      />

      {connectionsQ.isLoading ? (
        <ListSkeleton />
      ) : connectionsQ.isError && !connectionsQ.data ? (
        <ErrorState
          title="Could not load connections"
          description={adapterErrorMessage(connectionsQ.error)}
          onRetry={() => void connectionsQ.refetch()}
          retrying={connectionsQ.isFetching}
        />
      ) : hasConnections ? (
        <ConnectionList token={token} organizationId={org.id} connections={connections} catalog={catalogQ.data ?? []} justConnectedId={justConnectedId} />
      ) : (
        <section aria-labelledby="connect-first-title" className="space-y-6">
          <div className="flex items-start gap-4">
            {/* Clutter's block joined to yours: the connection. */}
            <AssembleInView className="mt-1 flex shrink-0">
              <IsoStack
                blocks={[
                  { x: 0, y: 2, accent: true },
                  { x: 2, y: 0 },
                ]}
                links={[[0, 1]]}
                accent="brand"
                size={18}
                assemble
                idle="lift"
              />
            </AssembleInView>
            <div className="space-y-1">
              <h2 id="connect-first-title" className="text-foreground text-base font-semibold">
                Connect your first cloud
              </h2>
              <p className="text-muted-foreground max-w-prose text-sm">
                A connection is a cloud account Clutter can build in. Choose a cloud to start; Clutter checks the account before it saves it.
              </p>
            </div>
          </div>
          <ConnectCloudFlow
            token={token}
            organizationId={org.id}
            initialConnector={openConnector ?? firstCloud}
            onConnected={(c) => setJustConnectedId(c.id)}
          />
        </section>
      )}

      <ConnectCloudSheet
        open={open}
        onOpenChange={onSheetOpenChange}
        organizationId={org.id}
        preferredConnector={sheetCloud}
        onConnected={setJustConnectedId}
      />
    </>
  );
}
