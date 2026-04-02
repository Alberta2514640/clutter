import { ExternalLink } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

interface AwsRoleSetupCardProps {
  accountName: string;
  defaultRegion: string;
  isGenerating: boolean;
  launchUrl: string | null;
  linkedAccountId: string | null;
  onAccountNameChange: (value: string) => void;
  onDefaultRegionChange: (value: string) => void;
  onGenerateStackUrl: () => Promise<void>;
}

const AWS_SETUP_GUIDE_URL =
  "https://clutter-templates-us-west-2-446fe866.s3.us-west-2.amazonaws.com/templates/pdf/AWS_Account_Link_Steps.pdf";

export default function AwsRoleSetupCard({ accountName, defaultRegion, isGenerating, launchUrl, linkedAccountId, onAccountNameChange, onDefaultRegionChange, onGenerateStackUrl }: AwsRoleSetupCardProps) {
  return (
    <Card className="h-full border-slate-800 bg-slate-950/40 shadow-none">
      <CardHeader>
        <CardTitle className="text-xl text-white">1. Open AWS setup</CardTitle>
        <CardDescription className="text-slate-400">Enter the account name first. Clutter will generate the CloudFormation launch URL and open AWS for you.</CardDescription>
      </CardHeader>
      <CardContent className="flex-1 space-y-5">
        <div className="grid gap-4 rounded-2xl border border-slate-800 bg-slate-900/60 p-4">
          <div className="space-y-2">
            <Label className="text-slate-200">AWS account name</Label>
            <Input value={accountName} onChange={(event) => onAccountNameChange(event.target.value)} placeholder="e.g. DemoAccount" className="border-slate-700 bg-slate-900/40 text-white placeholder:text-slate-500 focus-visible:ring-teal-500" />
            <p className="text-xs text-slate-500">This becomes the `accountName` query parameter for the setup-link endpoint.</p>
          </div>

          <div className="space-y-2">
            <Label className="text-slate-200">Deployment region</Label>
            <Select value={defaultRegion} onValueChange={onDefaultRegionChange}>
              <SelectTrigger className="w-full border-slate-700 bg-slate-900/40 text-white">
                <SelectValue placeholder="Select a region" />
              </SelectTrigger>
              <SelectContent className="border-slate-700 bg-slate-900 text-white">
                <SelectItem value="us-west-2">us-west-2</SelectItem>
                <SelectItem value="us-east-1">us-east-1</SelectItem>
                <SelectItem value="ca-central-1">ca-central-1</SelectItem>
                <SelectItem value="eu-west-1">eu-west-1</SelectItem>
              </SelectContent>
            </Select>
            <p className="text-xs text-slate-500">If omitted, the API defaults to `us-west-2`. Keeping it explicit is clearer for the user.</p>
          </div>

          <div className="flex flex-col gap-3 sm:flex-row">
            <Button type="button" className="bg-teal-500 font-semibold text-slate-950 hover:bg-teal-400" onClick={() => void onGenerateStackUrl()} disabled={isGenerating}>
              {isGenerating ? "Opening AWS setup..." : "Open AWS setup in CloudFormation"}
            </Button>
          </div>

          <div className="flex flex-col gap-3 rounded-xl border border-slate-800 bg-slate-950/50 px-4 py-3 text-sm text-slate-300 sm:flex-row sm:items-center sm:justify-between">
            <span>Need a walkthrough? Open the step-by-step AWS account linking guide.</span>
            <Button variant="ghost" size="sm" asChild className="text-slate-200 hover:bg-slate-800 hover:text-white">
              <a href={AWS_SETUP_GUIDE_URL} target="_blank" rel="noreferrer">
                View setup guide
                <ExternalLink className="h-4 w-4" />
              </a>
            </Button>
          </div>

          {linkedAccountId && (
            <div className="rounded-xl border border-teal-500/20 bg-teal-500/10 px-4 py-3 text-sm text-teal-50">
              Pending account link created with ID <span className="font-mono">{linkedAccountId}</span>.
            </div>
          )}

          {launchUrl && (
            <div className="flex flex-col gap-3 rounded-xl border border-slate-700/70 bg-slate-950/40 px-4 py-3 text-sm text-slate-300 sm:flex-row sm:items-center sm:justify-between">
              <span>AWS setup is ready. If the tab did not open, use the fallback action.</span>
              <Button type="button" variant="ghost" size="sm" className="text-slate-200 hover:bg-slate-800 hover:text-white" onClick={() => window.open(launchUrl, "_blank", "noopener,noreferrer")}>
                Open again
                <ExternalLink className="h-4 w-4" />
              </Button>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
