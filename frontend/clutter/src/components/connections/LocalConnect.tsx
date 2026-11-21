"use client";

import { Button } from "@/components/ui/button";
import { useLinkedAccounts } from "@/lib/features/catalog/hooks";
import { connectFailure } from "@/lib/features/connections/awsLink";
import { useConnections, useCreateConnection } from "@/lib/features/connections/hooks";
import type { Connection } from "@/lib/features/connections/types";
import { isLocalDevApi } from "@/lib/features/organization/api";
import { useLinkDevstack, useLinkFakeAccount } from "@/lib/features/organization/hooks";
import { apiErrorMessage } from "@/lib/features/user/session";
import { Laptop, Loader2 } from "lucide-react";
import { useState, type ReactNode } from "react";

// The local-only one-click connections (ADR-011). B and the devserver do not
// say whether this machine builds AWS on LocalStack or in real AWS, so these
// show in every development build that talks to the local gateway, and say
// when they cannot work. A production build drops them and their copy.

type Props = {
  token: string;
  organizationId: string;
  connector: string;
  // The cloud's short name, for the button ("Connect local AWS").
  cloudName: string;
  // The new connection's name (localConnectionName).
  name: string;
  // Called with the new connection.
  onConnected: (connection: Connection) => void;
};

// ConnectLocalAccount links a fake AWS account (A2's POST
// /dev/link-fake-account) and connects it in B, in one step. Deploys then run
// on LocalStack.
export function ConnectLocalAccount(props: Props & { region: string }) {
  // NODE_ENV stays inline so a production build drops the action and its copy.
  if (process.env.NODE_ENV !== "development" || !isLocalDevApi) return null;
  return <ConnectLocalAccountButton {...props} />;
}

function ConnectLocalAccountButton({ token, organizationId, connector, cloudName, name, region, onConnected }: Props & { region: string }) {
  const linkM = useLinkFakeAccount(token);
  const linkedQ = useLinkedAccounts(token, organizationId, connector);
  const createM = useCreateConnection(token);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const connect = async () => {
    setBusy(true);
    setError(null);
    let linking = true;
    try {
      const accountName = await linkM.mutateAsync(organizationId);
      linking = false;
      // B lists the account once A holds it; read the list again for its id.
      const accounts = (await linkedQ.refetch()).data ?? [];
      const account = accounts.find((a) => a.name === accountName);
      if (!account) {
        setError("The fake account was linked, but Clutter could not find it yet. Try again.");
        return;
      }
      const made = await createM.mutateAsync({ organizationId, data: { connector, name, region: account.region || region, linked_id: account.id } });
      onConnected(made);
    } catch (err) {
      setError(linking ? "Could not link the fake account. Check the gateway log." : connectFailure(err, true));
    } finally {
      setBusy(false);
    }
  };

  return (
    <LocalCard
      onClick={() => void connect()}
      busy={busy}
      error={error}
      label={`Connect local ${cloudName}`}
      text={
        <>
          Links a fake {cloudName} account and connects it in one click. Deploys then run on LocalStack, if it was running when Clutter started. It
          does not work while this machine is set up for real AWS.
        </>
      }
    />
  );
}

// UseLocalDevstack connects the OpenStack running on this machine (DevStack):
// the local gateway's POST /dev/link-devstack makes a new credential in the
// devstack VM and the connection in B; the secret never reaches the browser.
export function UseLocalDevstack(props: Props) {
  // NODE_ENV stays inline so a production build drops the button and its copy.
  if (process.env.NODE_ENV !== "development" || !isLocalDevApi) return null;
  return <UseLocalDevstackButton {...props} />;
}

function UseLocalDevstackButton({ token, organizationId, connector, cloudName, name, onConnected }: Props) {
  const linkM = useLinkDevstack(token);
  const connectionsQ = useConnections(token, organizationId);
  const [error, setError] = useState<string | null>(null);

  const connect = async () => {
    setError(null);
    try {
      const made = await linkM.mutateAsync({ organizationId, connector, name });
      // The route answers with the connection's name; the list has its id.
      const found = ((await connectionsQ.refetch()).data ?? []).find((c) => c.name === (made || name));
      if (!found) {
        setError("The connection was made, but Clutter could not find it yet. Reload the page.");
        return;
      }
      onConnected(found);
    } catch (err) {
      setError(apiErrorMessage(err) ?? "Could not reach the local gateway.");
    }
  };

  return (
    <LocalCard
      onClick={() => void connect()}
      busy={linkM.isPending}
      error={error}
      label={`Connect local ${cloudName}`}
      text={<>Uses the {cloudName} running on this machine (DevStack). One click, nothing to paste.</>}
    />
  );
}

function LocalCard({ onClick, busy, error, label, text }: { onClick: () => void; busy: boolean; error: string | null; label: string; text: ReactNode }) {
  return (
    <div className="border-border flex flex-col gap-3 rounded-lg border border-dashed p-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="min-w-0 space-y-1">
        <p className="micro-label text-foreground flex items-center gap-1.5">
          <Laptop className="text-link size-3.5" aria-hidden="true" />
          On this machine only
        </p>
        <p className="text-muted-foreground text-sm">{text}</p>
        {error && (
          <p role="alert" className="text-danger text-sm">
            {error}
          </p>
        )}
      </div>
      <Button type="button" variant="secondary" onClick={onClick} disabled={busy} className="shrink-0 self-start sm:self-auto">
        {busy && <Loader2 className="animate-spin" aria-hidden />}
        {busy ? "Connecting…" : label}
      </Button>
    </div>
  );
}
