import { Button } from "@/components/ui/button";
import type { OrganizationAwsAccount } from "@/lib/features/organization/types";

interface AwsExistingAccountCardProps {
  account: OrganizationAwsAccount;
  isDeleting: boolean;
  onRequestDelete: () => void;
}

const statusLabel = (status: string) => {
  switch (status) {
    case "complete":
      return "Linked";
    case "incomplete":
      return "Unfinished: waiting for the role ARN";
    case "revoked":
      return "Revoked";
    default:
      return status;
  }
};

// One AWS account link of the organization, linked or unfinished.
export default function AwsExistingAccountCard({ account, isDeleting, onRequestDelete }: AwsExistingAccountCardProps) {
  return (
    <li className="flex flex-col gap-3 rounded-xl border border-slate-800 bg-slate-900/70 px-4 py-3 text-sm text-slate-300 md:flex-row md:items-center md:justify-between">
      <div className="min-w-0 space-y-1">
        <p className="font-medium text-white">{account.account_name}</p>
        <p className={account.status === "complete" ? "text-teal-300" : "text-amber-300"}>{statusLabel(account.status)}</p>
        {account.role_arn && <p className="truncate font-mono text-xs text-slate-400">{account.role_arn}</p>}
      </div>
      <Button
        type="button"
        onClick={onRequestDelete}
        disabled={isDeleting}
        aria-label={`Remove the AWS account link ${account.account_name}`}
        variant="outline"
        className="shrink-0 border-red-800 bg-transparent text-red-300 hover:bg-red-950 hover:text-red-200">
        Remove link
      </Button>
    </li>
  );
}
