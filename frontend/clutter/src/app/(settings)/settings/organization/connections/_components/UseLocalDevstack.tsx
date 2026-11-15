"use client";

import { isLocalDevApi } from "@/lib/features/organization/api";
import { useLinkDevstack } from "@/lib/features/organization/hooks";
import { apiErrorMessage } from "@/lib/features/user/session";
import { Loader2 } from "lucide-react";

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
    <div className="flex flex-col gap-3 rounded-xl border border-teal-500/40 bg-teal-500/5 p-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="space-y-1">
        <p className="text-sm text-slate-100">Use the {cloudName} running on this machine. One click, nothing to paste.</p>
        {linkM.isError && (
          <p role="alert" className="text-sm text-red-300">
            {apiErrorMessage(linkM.error) ?? "Could not reach the local gateway."}
          </p>
        )}
      </div>
      <button
        type="button"
        onClick={() => linkM.mutate({ organizationId, connector, name }, { onSuccess: onLinked })}
        disabled={linkM.isPending}
        className="inline-flex shrink-0 items-center gap-2 rounded-lg bg-teal-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-teal-500 disabled:cursor-not-allowed disabled:opacity-60"
      >
        {linkM.isPending && <Loader2 className="h-4 w-4 animate-spin" aria-hidden />}
        {linkM.isPending ? "Connecting…" : `Connect local ${cloudName}`}
      </button>
    </div>
  );
}
