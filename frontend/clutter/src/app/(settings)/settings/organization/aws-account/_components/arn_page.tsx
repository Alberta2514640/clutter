"use client";

import { useState } from "react";
import Link from "next/link";
import { AlertCircle, ArrowLeft } from "lucide-react";

import ConfirmDialog from "@/components/common/ConfirmDialog";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { useDeleteOrganizationAccount, useCurrentOrganization, useOrganizationAccounts } from "@/lib/features/organization/hooks";
import { useMe } from "@/lib/features/user/hooks";
import AwsAccountHeader from "./AwsAccountHeader";
import AwsExistingAccountCard from "./AwsExistingAccountCard";
import AwsLinkFlow from "./AwsLinkFlow";

type DeleteTarget = {
  accountId: string;
  accountName: string;
};

const CONNECTIONS_PATH = "/settings/organization/connections";

// The organization's AWS account links: each one listed with Remove, and the
// two-step flow to link another. Connections shows the same flow inline.
export default function ArnPage() {
  const meQ = useMe();
  const token = meQ.data?.token ?? null;
  const currentOrg = useCurrentOrganization(token);
  const organizationId = currentOrg.organizationId ?? "";
  const accountsQ = useOrganizationAccounts(token, organizationId);
  const deleteAccount = useDeleteOrganizationAccount(token);

  const [deleteTarget, setDeleteTarget] = useState<DeleteTarget | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const accounts = accountsQ.data ?? [];
  const linkedAccounts = accounts.filter((account) => account.status === "complete");

  const handleDeleteAccount = async () => {
    if (!organizationId || !deleteTarget) return;
    setDeleteError(null);

    try {
      await deleteAccount.mutateAsync({
        organizationId,
        accountId: deleteTarget.accountId,
      });
    } catch {
      setDeleteError("Clutter could not remove the account link. Try again.");
      return;
    }
    setDeleteTarget(null);
  };

  return (
    <div className="space-y-6">
      <Link href={CONNECTIONS_PATH} className="inline-flex items-center gap-2 text-sm text-slate-300 hover:text-white">
        <ArrowLeft className="h-4 w-4" aria-hidden="true" />
        Back to Connections
      </Link>

      <AwsAccountHeader hasLinkedAccount={linkedAccounts.length > 0} />

      {accountsQ.isLoading ? (
        <p role="status" className="text-sm text-slate-400">
          Loading your AWS account links...
        </p>
      ) : accountsQ.isError ? (
        <Alert className="border-red-500/30 bg-red-500/10 text-red-100">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>Clutter could not load your AWS account links. Reload the page to try again.</AlertDescription>
        </Alert>
      ) : (
        accounts.length > 0 && (
          <section aria-labelledby="aws-links-heading" className="space-y-3">
            <h2 id="aws-links-heading" className="text-lg font-semibold text-white">
              AWS account links{currentOrg.organization ? ` in ${currentOrg.organization.name}` : ""}
            </h2>
            <ul className="space-y-3">
              {accounts.map((account) => (
                <AwsExistingAccountCard
                  key={account.id}
                  account={account}
                  isDeleting={deleteAccount.isPending}
                  onRequestDelete={() => {
                    setDeleteError(null);
                    setDeleteTarget({ accountId: account.id, accountName: account.account_name });
                  }}
                />
              ))}
            </ul>
          </section>
        )
      )}

      {organizationId && (
        <Card className="rounded-2xl border border-slate-800 bg-slate-900/70 shadow-xl">
          <CardContent className="px-6 py-6">
            <AwsLinkFlow
              token={token}
              organizationId={organizationId}
              afterLink={
                <>
                  <p>On the Connections page, use it to deploy your diagrams to AWS.</p>
                  <Button asChild className="bg-teal-500 font-semibold text-slate-950 hover:bg-teal-400">
                    <Link href={CONNECTIONS_PATH}>Back to Connections</Link>
                  </Button>
                </>
              }
            />
          </CardContent>
        </Card>
      )}

      {deleteTarget && (
        <ConfirmDialog
          title="Remove AWS account link"
          confirmLabel="Remove link"
          pendingLabel="Removing…"
          isPending={deleteAccount.isPending}
          error={deleteError}
          onCancel={() => setDeleteTarget(null)}
          onConfirm={() => void handleDeleteAccount()}>
          <p>
            Remove <span className="font-semibold text-white">&quot;{deleteTarget.accountName}&quot;</span> from this organization? To use this AWS account again, you link it again.
          </p>
          <p>The role stays in your AWS account until you delete its stack there.</p>
        </ConfirmDialog>
      )}
    </div>
  );
}
