import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import type { OrganizationAwsAccount } from "@/lib/features/organization/types";

interface AwsExistingAccountCardProps {
  account: OrganizationAwsAccount;
}

export default function AwsExistingAccountCard({ account }: AwsExistingAccountCardProps) {
  return (
    <Card className="border-slate-800 bg-slate-950/40 shadow-none">
      <CardHeader>
        <CardTitle className="text-xl text-white">Linked AWS account</CardTitle>
        <CardDescription className="text-slate-400">
          This organization already has an AWS account link configured.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3 text-sm text-slate-300">
        <div className="flex items-center justify-between rounded-xl border border-slate-800 bg-slate-900/70 px-4 py-3">
          <span className="text-slate-400">Account name</span>
          <span className="font-medium text-white">{account.account_name}</span>
        </div>
        <div className="flex items-center justify-between rounded-xl border border-slate-800 bg-slate-900/70 px-4 py-3">
          <span className="text-slate-400">Status</span>
          <span className="font-medium text-white">{account.status}</span>
        </div>
        <div className="flex items-center justify-between rounded-xl border border-slate-800 bg-slate-900/70 px-4 py-3">
          <span className="text-slate-400">Role ARN</span>
          <span className="max-w-[20rem] truncate font-mono text-white">{account.role_arn ?? "Not submitted yet"}</span>
        </div>
      </CardContent>
    </Card>
  );
}
