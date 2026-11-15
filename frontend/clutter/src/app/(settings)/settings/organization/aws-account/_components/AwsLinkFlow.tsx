"use client";

import { AlertCircle, CheckCircle2 } from "lucide-react";
import { useState, type ReactNode } from "react";

import LinkFakeAccount from "@/components/common/LinkFakeAccount";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { useCreateCloudFormationStackUrl, useOrganizationAccounts, useUpdateOrganizationAccount } from "@/lib/features/organization/hooks";
import { awsAccountNameProblem, awsRoleArnExample, awsRoleArnPattern } from "@/lib/features/organization/rules";
import type { CloudFormationStackUrlResponse } from "@/lib/features/organization/types";
import { ApiError, apiErrorMessage } from "@/lib/features/user/session";
import AwsRoleSetupCard from "./AwsRoleSetupCard";
import AwsRoleVerificationForm from "./AwsRoleVerificationForm";

export type LinkStatus = "idle" | "creating-template-link" | "awaiting-role-arn" | "saving" | "submitted" | "error";

type Props = {
  token: string | null;
  organizationId: string;
  // Called once an account is linked, with its name.
  onLinked?: (accountName: string) => void;
  // What the success notice says to do next.
  afterLink: ReactNode;
  // The two steps side by side from this breakpoint up.
  wide?: "lg" | "xl";
};

// AwsLinkFlow links an AWS account in two steps: open AWS to create the role
// (a CloudFormation stack Clutter fills in), then paste the role's ARN. It
// resumes the organization's first unfinished link. Used by the AWS account
// page and inline on Connections. Locally, while the organization has no
// linked account, it also offers the Link fake account button.
export default function AwsLinkFlow({ token, organizationId, onLinked, afterLink, wide = "lg" }: Props) {
  const createStackUrl = useCreateCloudFormationStackUrl(token);
  const accountsQ = useOrganizationAccounts(token, organizationId);
  const updateAccount = useUpdateOrganizationAccount(token);

  const [accountName, setAccountName] = useState("");
  const [roleArn, setRoleArn] = useState("");
  const [status, setStatus] = useState<LinkStatus>("idle");
  const [error, setError] = useState("");
  const [started, setStarted] = useState<CloudFormationStackUrlResponse | null>(null);
  const [linkedName, setLinkedName] = useState<string | null>(null);

  const accounts = accountsQ.data ?? [];
  // A link started here and since removed (on the AWS account page) no longer counts.
  const stackData = started && (accountsQ.isFetching || accounts.some((a) => a.id === started.account_id)) ? started : null;
  // The flow resumes the first unfinished link; the AWS account page lists the others.
  const pendingAccount = accounts.find((account) => account.status === "incomplete") ?? null;
  const activePendingAccountId = stackData?.account_id ?? pendingAccount?.id ?? null;
  const effectiveAccountName = accountName.trim() || pendingAccount?.account_name || "";
  const effectiveRoleArn = roleArn.trim() || pendingAccount?.role_arn || "";

  const fail = (message: string) => {
    setStatus("error");
    setError(message);
  };

  const handleGenerateStackUrl = async () => {
    if (!token || !organizationId) {
      fail("Your organization is still loading. Try again in a moment.");
      return;
    }
    if (!accountName.trim()) {
      fail("Enter a name for this AWS account first.");
      return;
    }
    const nameProblem = awsAccountNameProblem(accountName);
    if (nameProblem) {
      fail(`That account name will not work. ${nameProblem}`);
      return;
    }

    setError("");
    setLinkedName(null);
    setStatus("creating-template-link");
    const popup = typeof window !== "undefined" ? window.open("", "_blank") : null;

    try {
      const response = await createStackUrl.mutateAsync({ organizationId, accountName: accountName.trim() });
      setStarted(response);
      setStatus("awaiting-role-arn");
      if (popup) {
        popup.location.href = response.url;
        popup.focus();
      }
    } catch (err) {
      popup?.close();
      setStarted(null);
      // A refused request (400) says why; anything else is worth a retry.
      const reason = err instanceof ApiError && err.status === 400 ? apiErrorMessage(err) : null;
      fail(reason ? `Clutter could not prepare the AWS setup: ${reason}` : "Clutter could not prepare the AWS setup. Try again.");
    }
  };

  const handleSave = async () => {
    if (!organizationId || !activePendingAccountId) {
      fail("Open AWS in step 1 first.");
      return;
    }
    if (!awsRoleArnPattern.test(effectiveRoleArn)) {
      fail(`That is not the role Clutter's AWS setup creates. Copy the role ARN from the stack's Outputs tab. It looks like ${awsRoleArnExample}.`);
      return;
    }

    setError("");
    setStatus("saving");

    try {
      await updateAccount.mutateAsync({ organizationId, accountId: activePendingAccountId, data: { role_arn: effectiveRoleArn } });
      setLinkedName(effectiveAccountName);
      setAccountName("");
      setRoleArn("");
      setStarted(null);
      setStatus("submitted");
      onLinked?.(effectiveAccountName);
    } catch (err) {
      // The backend refuses an ARN whose role was not made by the stack Clutter opened for this link.
      fail(
        err instanceof ApiError && err.status === 400
          ? "This role does not belong to the AWS setup Clutter opened for this account. Copy the role ARN from the Outputs tab of that stack, or remove this link and start again."
          : "Clutter could not link the account. Try again.",
      );
    }
  };

  // The local-only fake link ends like a real one: the notice, then onLinked.
  const handleFakeLinked = (name: string) => {
    setError("");
    setAccountName("");
    setRoleArn("");
    setStarted(null);
    setLinkedName(name);
    setStatus("submitted");
    onLinked?.(name);
  };
  // The dev route links a fake account only while none is linked.
  const offerFakeLink = accountsQ.isSuccess && !accounts.some((account) => account.status === "complete");

  const hasPendingFlow = !!pendingAccount || !!stackData;
  const currentStatus: LinkStatus = (status === "idle" || status === "submitted") && pendingAccount ? "awaiting-role-arn" : status;

  return (
    <div className="space-y-4">
      {status === "submitted" && linkedName && (
        <Alert role="status" className="border-teal-500/30 bg-teal-500/10 text-teal-50">
          <CheckCircle2 className="h-4 w-4 text-teal-300" />
          <AlertTitle>AWS account linked</AlertTitle>
          <AlertDescription className="space-y-3 text-teal-50/90">
            <p>{linkedName} is linked.</p>
            {afterLink}
          </AlertDescription>
        </Alert>
      )}

      {status === "error" && error && (
        <Alert className="border-red-500/30 bg-red-500/10 text-red-100">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>The account is not linked yet</AlertTitle>
          <AlertDescription className="text-red-100/90">{error}</AlertDescription>
        </Alert>
      )}

      {offerFakeLink && <LinkFakeAccount token={token} organizationId={organizationId} onLinked={handleFakeLinked} variant="inline" />}

      <div className={`grid gap-6 ${wide === "xl" ? "xl:grid-cols-2" : "lg:grid-cols-2"}`}>
        <AwsRoleSetupCard
          accountName={effectiveAccountName}
          isGenerating={createStackUrl.isPending}
          launchUrl={stackData?.url ?? null}
          setupStarted={hasPendingFlow}
          onAccountNameChange={setAccountName}
          onGenerateStackUrl={handleGenerateStackUrl}
        />
        <AwsRoleVerificationForm
          accountName={effectiveAccountName}
          isReadyForArn={hasPendingFlow}
          roleArn={effectiveRoleArn}
          status={currentStatus}
          onRoleArnChange={setRoleArn}
          onSave={handleSave}
        />
      </div>
    </div>
  );
}
