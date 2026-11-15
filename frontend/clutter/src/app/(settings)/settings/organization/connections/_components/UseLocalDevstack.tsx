"use client";

import { isLocalDevApi } from "@/lib/features/organization/api";
import { useLinkDevstack } from "@/lib/features/organization/hooks";
import { apiErrorMessage } from "@/lib/features/user/session";
import LocalConnectCard from "./LocalConnectCard";

type Props = {
  token: string;
  organizationId: string;
  connector: string;
  // The cloud's short name, for the button ("Connect local OpenStack").
  cloudName: string;
  // The new connection's name (localConnectionName).
  name: string;
  // Called with the new connection's name.
  onLinked: (connectionName: string) => void;
};

// UseLocalDevstack is the local-only "Connect local <cloud>" button, the
// twin of Connect local AWS (ADR-011). The local dev gateway's POST
// /dev/link-devstack makes a new credential in the devstack VM and the
// connection in B; the secret never reaches the browser. It renders nothing
// unless this is a development build talking to the local gateway.
export default function UseLocalDevstack(props: Props) {
  // NODE_ENV stays inline so a production build drops the button and its copy.
  if (process.env.NODE_ENV !== "development" || !isLocalDevApi) return null;
  return <UseLocalDevstackButton {...props} />;
}

function UseLocalDevstackButton({ token, organizationId, connector, cloudName, name, onLinked }: Props) {
  const linkM = useLinkDevstack(token);

  return (
    <LocalConnectCard
      cloudName={cloudName}
      onClick={() => linkM.mutate({ organizationId, connector, name }, { onSuccess: onLinked })}
      busy={linkM.isPending}
      error={linkM.isError ? (apiErrorMessage(linkM.error) ?? "Could not reach the local gateway.") : null}
    />
  );
}
