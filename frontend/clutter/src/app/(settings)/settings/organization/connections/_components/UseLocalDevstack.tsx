"use client";

import { isLocalDevApi } from "@/lib/features/organization/api";
import { useLinkDevstack } from "@/lib/features/organization/hooks";
import { apiErrorMessage } from "@/lib/features/user/session";
import { Loader2 } from "lucide-react";

type Props = {
  token: string;
  organizationId: string;
  connector: string;
  // Called with the new connection's name.
  onLinked: (connectionName: string) => void;
};

// UseLocalDevstack is the local-only "Use this machine's DevStack" button, the
// twin of Link fake account (ADR-011). The local dev gateway's POST
// /dev/link-devstack makes a new credential in the devstack VM and the
// connection in B; the secret never reaches the browser. It renders nothing
// unless this is a development build talking to the local gateway.
export default function UseLocalDevstack(props: Props) {
  // NODE_ENV stays inline so a production build drops the button and its copy.
  if (process.env.NODE_ENV !== "development" || !isLocalDevApi) return null;
  return <UseLocalDevstackButton {...props} />;
}

function UseLocalDevstackButton({ token, organizationId, connector, onLinked }: Props) {
  const linkM = useLinkDevstack(token);

  return (
    <div className="flex flex-col gap-3 rounded-xl border border-dashed border-amber-500/40 bg-amber-500/5 p-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="space-y-1">
        <p className="text-sm text-amber-100">On this machine only: connect the local DevStack in one step.</p>
        {linkM.isError && (
          <p role="alert" className="text-sm text-red-300">
            {apiErrorMessage(linkM.error) ?? "Could not reach the local gateway."}
          </p>
        )}
      </div>
      <button
        type="button"
        onClick={() => linkM.mutate({ organizationId, connector }, { onSuccess: onLinked })}
        disabled={linkM.isPending}
        className="inline-flex shrink-0 items-center gap-2 rounded-lg border border-amber-500/40 px-4 py-2 text-sm font-medium text-amber-200 transition hover:bg-amber-500/10 disabled:cursor-not-allowed disabled:opacity-60"
      >
        {linkM.isPending && <Loader2 className="h-4 w-4 animate-spin" aria-hidden />}
        {linkM.isPending ? "Connecting…" : "Use this machine's DevStack"}
      </button>
    </div>
  );
}
