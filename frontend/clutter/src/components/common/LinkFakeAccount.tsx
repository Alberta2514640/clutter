"use client";

import { isLocalDevApi } from "@/lib/features/organization/api";
import { useLinkFakeAccount } from "@/lib/features/organization/hooks";

type Props = {
  token: string | null;
  organizationId: string | null;
  // Called with the linked account's name: a new fake one, or the one the
  // organization already had.
  onLinked?: (accountName: string) => void;
  // "popup" is the bare button for A's "AWS account required" popup; "inline"
  // adds a short note, for the AWS link flow.
  variant: "popup" | "inline";
};

// LinkFakeAccount is the local-only Link fake account button (ADR-011). It
// links an AWS account with a role in no real account, through the local dev
// gateway's POST /dev/link-fake-account. It renders nothing unless this is a
// development build talking to the local gateway.
export default function LinkFakeAccount(props: Props) {
  // NODE_ENV stays inline so a production build drops the button and its copy.
  if (process.env.NODE_ENV !== "development" || !isLocalDevApi) return null;
  return <LinkFakeAccountButton {...props} />;
}

function LinkFakeAccountButton({ token, organizationId, onLinked, variant }: Props) {
  const linkM = useLinkFakeAccount(token);

  const button = (
    <button
      type="button"
      onClick={() => {
        if (organizationId) linkM.mutate(organizationId, { onSuccess: (accountName) => onLinked?.(accountName) });
      }}
      disabled={linkM.isPending || !token || !organizationId}
      className="border-warning/40 text-warning hover:bg-warning/10 rounded-lg border px-4 py-2 text-sm font-medium transition disabled:cursor-not-allowed disabled:opacity-60"
    >
      {linkM.isPending ? "Linking…" : "Link fake account"}
    </button>
  );
  const failed = linkM.isError && (
    <p role="alert" className="text-danger text-sm">
      Could not link a fake account. Check the gateway log.
    </p>
  );

  if (variant === "popup") {
    return (
      <div className="flex flex-col items-end gap-2">
        {button}
        {failed}
      </div>
    );
  }

  return (
    <div className="border-warning/40 bg-warning/5 flex flex-col gap-3 rounded-xl border border-dashed p-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="space-y-1">
        <p className="text-foreground text-sm">
          On this machine only: link a fake AWS account instead of opening AWS.
        </p>
        <p className="text-muted-foreground text-xs">
          AWS deploys then run on LocalStack, if it was running when Clutter started.
        </p>
        {failed}
      </div>
      <div className="shrink-0">{button}</div>
    </div>
  );
}
