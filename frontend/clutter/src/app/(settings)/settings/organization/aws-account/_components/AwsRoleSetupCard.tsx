import { ExternalLink } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { AWS_ACCOUNT_NAME_MAX, awsAccountNameHint } from "@/lib/features/organization/rules";

interface AwsRoleSetupCardProps {
  accountName: string;
  isGenerating: boolean;
  launchUrl: string | null;
  setupStarted: boolean;
  onAccountNameChange: (value: string) => void;
  onGenerateStackUrl: () => Promise<void>;
}

export default function AwsRoleSetupCard({ accountName, isGenerating, launchUrl, setupStarted, onAccountNameChange, onGenerateStackUrl }: AwsRoleSetupCardProps) {
  return (
    <Card className="h-full border-slate-800 bg-slate-950/40 shadow-none">
      <CardHeader>
        <CardTitle className="text-xl text-white">1. Create the role in AWS</CardTitle>
        <CardDescription className="text-slate-400">
          Name this account, then open AWS. Sign in there and create the stack Clutter fills in for you. When it finishes, copy the role ARN from the stack&apos;s Outputs tab.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex-1 space-y-5">
        <div className="grid gap-4 rounded-2xl border border-slate-800 bg-slate-900/60 p-4">
          <div className="space-y-2">
            <Label htmlFor="aws-account-name" className="text-slate-200">
              Account name
            </Label>
            <Input
              id="aws-account-name"
              value={accountName}
              onChange={(event) => onAccountNameChange(event.target.value)}
              placeholder="e.g. Production"
              maxLength={AWS_ACCOUNT_NAME_MAX}
              aria-describedby="aws-account-name-hint"
              disabled={setupStarted}
              className="border-slate-700 bg-slate-900/40 text-white placeholder:text-slate-500 focus-visible:ring-teal-500"
            />
            <p id="aws-account-name-hint" className="text-xs text-slate-500">
              A name you will recognise on the Connections page. {awsAccountNameHint}
            </p>
          </div>

          <p className="text-xs text-slate-500">
            AWS opens in the Canada (Central) region, ca-central-1. The role works in every region.
          </p>

          <div className="flex flex-col gap-3 sm:flex-row">
            <Button type="button" className="bg-teal-500 font-semibold text-slate-950 hover:bg-teal-400" onClick={() => void onGenerateStackUrl()} disabled={isGenerating || setupStarted}>
              {isGenerating ? "Opening AWS..." : "Open AWS"}
            </Button>
          </div>

          {launchUrl && (
            <div className="flex flex-col gap-3 rounded-xl border border-slate-700/70 bg-slate-950/40 px-4 py-3 text-sm text-slate-300 sm:flex-row sm:items-center sm:justify-between">
              <span>AWS opened in a new tab. If it did not open, open it again here.</span>
              <Button type="button" variant="ghost" size="sm" className="text-slate-200 hover:bg-slate-800 hover:text-white" onClick={() => window.open(launchUrl, "_blank", "noopener,noreferrer")}>
                Open AWS again
                <ExternalLink className="h-4 w-4" aria-hidden="true" />
              </Button>
            </div>
          )}

          {setupStarted && !launchUrl && (
            <p className="rounded-xl border border-teal-500/20 bg-teal-500/10 px-4 py-3 text-sm text-teal-50">
              You already started this setup in AWS. Paste the role ARN in step 2, or remove the unfinished link on the AWS account links page to start again.
            </p>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
