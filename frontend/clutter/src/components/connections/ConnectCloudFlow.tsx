"use client";

import CloudMark from "@/components/clouds/CloudMark";
import ErrorState from "@/components/common/ErrorState";
import { Button } from "@/components/ui/button";
import { RadioCard, RadioCardDescription, RadioCardGroup, RadioCardTitle } from "@/components/ui/radio-card";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "@/components/ui/sonner";
import { adapterApiConfigured, adapterErrorMessage, NOT_CONFIGURED_MESSAGE } from "@/lib/adapterApi";
import { cloudLook } from "@/lib/features/catalog/display";
import { useCatalog } from "@/lib/features/catalog/hooks";
import type { CatalogEntry } from "@/lib/features/catalog/types";
import { cloudSummary, offeredClouds, preferredCloud } from "@/lib/features/connections/display";
import type { Connection } from "@/lib/features/connections/types";
import { cn } from "@/lib/utils";
import { CheckCircle2 } from "lucide-react";
import { useId, useState, type ComponentType } from "react";
import AwsConnectForm from "./AwsConnectForm";
import CredentialsForm from "./CredentialsForm";
import { Notice } from "./Field";

type FormProps = { token: string; organizationId: string; entry: CatalogEntry; onConnected: (c: Connection) => void; narrow?: boolean };

// A cloud whose accounts are linked outside B has its own link flow. AWS links
// through A's CloudFormation flow; every other cloud gets the form its
// credential spec describes.
const LINK_FLOWS: Record<string, ComponentType<FormProps>> = { aws: AwsConnectForm };

type Props = {
  token: string;
  organizationId: string;
  // The cloud the flow opens on, when this build offers it.
  initialConnector?: string | null;
  // Called once a connection is made and checked.
  onConnected?: (connection: Connection) => void;
  // Shown after a connection is made, beside "Connect another cloud" (the
  // sheet's Done). Without it the flow goes back to choosing a cloud.
  onDone?: () => void;
  // One column, for the side sheet.
  narrow?: boolean;
};

// ConnectCloudFlow is the one way to connect a cloud: choose the cloud, then
// fill its form. The Connections page shows it inline when the team has no
// connection yet, and ConnectCloudSheet holds it everywhere else.
export default function ConnectCloudFlow({ token, organizationId, initialConnector, onConnected, onDone, narrow = false }: Props) {
  const labelId = useId();
  const catalogQ = useCatalog(token);
  const [picked, setPicked] = useState<string | null>(null);
  const [made, setMade] = useState<Connection | null>(null);

  if (!adapterApiConfigured) return <Notice tone="error">{NOT_CONFIGURED_MESSAGE}</Notice>;

  if (catalogQ.isLoading) {
    return (
      <div role="status" aria-label="Loading the clouds Clutter offers" className={cn("grid gap-3", !narrow && "sm:grid-cols-2 lg:grid-cols-3")}>
        {[0, 1, 2].map((i) => (
          <Skeleton key={i} className="h-[74px] rounded-xl" />
        ))}
      </div>
    );
  }
  if (catalogQ.isError && !catalogQ.data) {
    return (
      <ErrorState
        variant="inline"
        description={`Could not load the clouds Clutter offers. ${adapterErrorMessage(catalogQ.error)}`}
        onRetry={() => void catalogQ.refetch()}
        retrying={catalogQ.isFetching}
      />
    );
  }

  // NODE_ENV stays inline so a production build never offers a test cloud.
  const offered = offeredClouds(catalogQ.data ?? [], process.env.NODE_ENV === "development");
  if (offered.length === 0) return <Notice tone="info">Clutter offers no clouds yet.</Notice>;

  const connector = picked ?? preferredCloud(offered, initialConnector);
  const entry = offered.find((e) => e.manifest.connector === connector) ?? null;
  const Form = entry ? (LINK_FLOWS[entry.manifest.connector] ?? CredentialsForm) : null;

  const connected = (c: Connection) => {
    toast.success(`Connected ${c.name}`, { description: "Clutter checked it and it works." });
    setMade(c);
    onConnected?.(c);
  };

  if (made) {
    return (
      <div role="status" className="border-success/30 bg-success/5 space-y-4 rounded-xl border p-5 animate-fade-up">
        <div className="flex items-start gap-3">
          <CheckCircle2 className="text-success mt-0.5 size-5 shrink-0 animate-check" aria-hidden="true" />
          <div className="space-y-1">
            <p className="text-foreground font-semibold">Connected {made.name}</p>
            <p className="text-muted-foreground text-sm">Clutter checked it and it works. Every diagram in this team can deploy to it now.</p>
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          {onDone && (
            <Button type="button" onClick={onDone}>
              Done
            </Button>
          )}
          <Button
            type="button"
            variant={onDone ? "outline" : "primary"}
            onClick={() => {
              setMade(null);
              setPicked(null);
            }}>
            Connect another cloud
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="space-y-2.5">
        <p id={labelId} className="text-foreground text-sm font-medium">
          Choose a cloud
        </p>
        <RadioCardGroup
          aria-labelledby={labelId}
          value={connector ?? ""}
          onValueChange={setPicked}
          className={cn(!narrow && "sm:grid-cols-2 lg:grid-cols-3")}>
          {offered.map((c) => (
            <RadioCard key={c.manifest.connector} value={c.manifest.connector} className="gap-3 p-3.5">
              <CloudMark cloud={cloudLook(c.manifest)} size={28} />
              <span className="min-w-0">
                <RadioCardTitle>{c.manifest.title}</RadioCardTitle>
                <RadioCardDescription className="text-xs">{cloudSummary(c)}</RadioCardDescription>
              </span>
            </RadioCard>
          ))}
        </RadioCardGroup>
      </div>

      {entry && Form && (
        <div className={narrow ? "border-t pt-6" : "surface-card p-5 md:p-6"}>
          <Form key={entry.manifest.connector} token={token} organizationId={organizationId} entry={entry} onConnected={connected} narrow={narrow} />
        </div>
      )}
    </div>
  );
}
