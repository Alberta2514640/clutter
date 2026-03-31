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

export default function AwsRoleSetupCard({ accountName, defaultRegion, isGenerating, launchUrl, linkedAccountId, onAccountNameChange, onDefaultRegionChange, onGenerateStackUrl }: AwsRoleSetupCardProps) {
  return (
    <>
      <Card className="border-slate-800 bg-slate-950/40 shadow-none">
        <CardHeader>
          <CardTitle className="text-xl text-white">1. Why Clutter needs this role</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-3 text-sm text-slate-300">
          <div className="rounded-xl border border-slate-800 bg-slate-900/70 p-4">Clutter assumes a cross-account IAM role only when running deployment actions for this workspace.</div>
          <div className="rounded-xl border border-slate-800 bg-slate-900/70 p-4">No AWS access key IDs, secret keys, or session tokens are collected in the browser.</div>
          <div className="rounded-xl border border-slate-800 bg-slate-900/70 p-4">You can keep permissions constrained to the resources your deployment engine actually manages.</div>
        </CardContent>
      </Card>

      <Card className="border-slate-800 bg-slate-950/40 shadow-none">
        <CardHeader>
          <CardTitle className="text-xl text-white">2. Generate the AWS setup link</CardTitle>
          <CardDescription className="text-slate-400">Enter the account name first. Clutter will request a CloudFormation launch URL tied to this organization.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
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
                {isGenerating ? "Generating setup link..." : "Generate CloudFormation link"}
              </Button>
              <Button type="button" variant="outline" className="border-slate-700 bg-slate-900/40 text-slate-200 hover:bg-slate-800" onClick={() => launchUrl && window.open(launchUrl, "_blank", "noopener,noreferrer")} disabled={!launchUrl}>
                Open AWS Console
                <ExternalLink className="h-4 w-4" />
              </Button>
            </div>

            {linkedAccountId && (
              <div className="rounded-xl border border-teal-500/20 bg-teal-500/10 px-4 py-3 text-sm text-teal-50">
                Pending account link created with ID <span className="font-mono">{linkedAccountId}</span>.
              </div>
            )}

            {launchUrl && <div className="rounded-xl border border-slate-700/70 bg-slate-950/40 px-4 py-3 text-sm text-slate-300">CloudFormation URL is ready. Open AWS, create the stack, then return here with the Role ARN.</div>}
          </div>

          <div className="grid gap-3 rounded-2xl border border-slate-800 bg-slate-900/60 p-4 text-sm text-slate-300">
            <div className="grid gap-2 text-slate-400">
              <p>1. Generate the setup link after entering the AWS account name.</p>
              <p>2. Open the AWS Console using the returned CloudFormation launch URL.</p>
              <p>3. Copy the created Role ARN from AWS, then paste it into the form on the right.</p>
            </div>

            <div className="rounded-xl border border-slate-700/70 bg-slate-950/40 p-4 text-slate-400">
              The CloudFormation link already contains the setup details generated by us. For this flow, you only need to open AWS from this link and return with the Role ARN.
            </div>
          </div>
        </CardContent>
      </Card>
    </>
  );
}
