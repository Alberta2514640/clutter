import { ArrowRight } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { LinkStatus } from "./arn_page";

interface AwsRoleVerificationFormProps {
  accountName: string;
  defaultRegion: string;
  isReadyForArn: boolean;
  roleArn: string;
  status: LinkStatus;
  onDefaultRegionChange: (value: string) => void;
  onRoleArnChange: (value: string) => void;
  onSave: () => Promise<void>;
  onVerify: () => Promise<void>;
}

export default function AwsRoleVerificationForm({ accountName, defaultRegion, isReadyForArn, roleArn, status, onDefaultRegionChange, onRoleArnChange, onSave, onVerify }: AwsRoleVerificationFormProps) {
  return (
    <Card className="h-full border-slate-800 bg-slate-950/40 shadow-none">
      <CardHeader>
        <CardTitle className="text-xl text-white">2. Paste role details and submit</CardTitle>
        <CardDescription className="text-slate-400">After you finish in AWS, paste the Role ARN here and submit it. Format validation is optional.</CardDescription>
      </CardHeader>
      <CardContent className="flex-1">
        <form
          className="space-y-5"
          onSubmit={(event) => {
            event.preventDefault();
            void onSave();
          }}
        >
          <div className="space-y-2">
            <Label className="text-slate-200">AWS account name</Label>
            <Input value={accountName} readOnly className="border-slate-700 bg-slate-950/60 text-slate-300" />
            <p className="text-xs text-slate-500">This comes from the setup-link step and is sent to the backend before the ARN step is unlocked.</p>
          </div>

          <div className="space-y-2">
            <Label className="text-slate-200">Role ARN</Label>
            <Input
              value={roleArn}
              onChange={(event) => onRoleArnChange(event.target.value)}
              placeholder="arn:aws:iam::123456789012:role/ClutterDeployRole"
              className="border-slate-700 bg-slate-900/40 font-mono text-white placeholder:text-slate-500 focus-visible:ring-teal-500"
              disabled={!isReadyForArn}
            />
            <p className="text-xs text-slate-500">Submitting this sends `role_arn` to the backend account endpoint.</p>
          </div>

          <div className="space-y-2">
            <Label className="text-slate-200">Default region</Label>
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
          </div>

          <div className="flex flex-col gap-3 pt-2">
            <Button type="submit" className="w-full bg-teal-500 font-semibold text-slate-950 hover:bg-teal-400" disabled={!isReadyForArn || status === "verifying" || status === "saving"}>
              {status === "saving" ? "Submitting ARN..." : "Submit ARN"}
              <ArrowRight className="h-4 w-4" />
            </Button>
            <Button type="button" variant="outline" className="w-full border-slate-700 bg-slate-900/40 text-slate-100 hover:bg-slate-800" onClick={() => void onVerify()} disabled={!isReadyForArn || status === "verifying" || status === "saving"}>
              {status === "verifying" ? "Validating ARN..." : "Optional: Validate ARN format"}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
