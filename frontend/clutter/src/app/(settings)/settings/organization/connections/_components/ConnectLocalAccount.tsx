"use client";

import { adapterErrorMessage } from "@/lib/adapterApi";
import { useLinkedAccounts } from "@/lib/features/catalog/hooks";
import { useCreateConnection } from "@/lib/features/connections/hooks";
import { isLocalDevApi } from "@/lib/features/organization/api";
import { useLinkFakeAccount } from "@/lib/features/organization/hooks";
import { useState } from "react";
import LocalConnectCard from "./LocalConnectCard";

type Props = {
  token: string;
  organizationId: string;
  connector: string;
  // The cloud's short name, for the button ("Connect local AWS").
  cloudName: string;
  // The new connection's name (localConnectionName) and its region.
  name: string;
  region: string;
  // Called with the new connection's name.
  onAdded: (connectionName: string) => void;
};

// ConnectLocalAccount is the local-only "Connect local AWS" action: in one
// step it links the fake account (A2's POST /dev/link-fake-account, ADR-011)
// and connects it in B from the organization's linked accounts. Deploys then
// run on LocalStack. It renders nothing unless this is a development build
// talking to the local gateway.
export default function ConnectLocalAccount(props: Props) {
  // NODE_ENV stays inline so a production build drops the action and its copy.
  if (process.env.NODE_ENV !== "development" || !isLocalDevApi) return null;
  return <ConnectLocalAccountButton {...props} />;
}

function ConnectLocalAccountButton({ token, organizationId, connector, cloudName, name, region, onAdded }: Props) {
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
        setError("The local account was linked, but Clutter could not find it yet. Try again.");
        return;
      }
      const made = await createM.mutateAsync({
        organizationId,
        data: { connector, name, region: account.region || region, linked_id: account.id },
      });
      onAdded(made.name);
    } catch (err) {
      setError(linking ? "Could not link the local account. Check the gateway log." : adapterErrorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return <LocalConnectCard cloudName={cloudName} onClick={() => void connect()} busy={busy} error={error} />;
}
