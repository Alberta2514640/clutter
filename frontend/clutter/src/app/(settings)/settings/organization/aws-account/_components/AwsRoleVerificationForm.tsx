import { ArrowRight } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import type { LinkStatus } from "./arn_page";

interface AwsRoleVerificationFormProps {
  accountName: string;
  defaultRegion: string;
  isReadyForArn: boolean;
  notes: string;
  roleArn: string;
  status: LinkStatus;
  onDefaultRegionChange: (value: string) => void;
  onNotesChange: (value: string) => void;
  onRoleArnChange: (value: string) => void;
  onSave: () => Promise<void>;
  onVerify: () => Promise<void>;
}

export default function AwsRoleVerificationForm({
  accountName,
  defaultRegion,
  isReadyForArn,
  notes,
  roleArn,
  status,
  onDefaultRegionChange,
  onNotesChange,
  onRoleArnChange,
  onSave,
  onVerify,
}: AwsRoleVerificationFormProps) {
  return (
    <Card className="border-slate-800 bg-slate-950/40 shadow-none">
      <CardHeader>
        <CardTitle className="text-xl text-white">3. Paste role details and submit</CardTitle>
        <CardDescription className="text-slate-400">Once the CloudFormation link has been generated and completed in AWS, paste the Role ARN here. The first button only validates format in the browser.</CardDescription>
      </CardHeader>
      <CardContent>
        <form
          className="space-y-5"
          onSubmit={(event) => {
            event.preventDefault();
            void onVerify();
          }}
        >
          <div className="space-y-2">
            <Label className="text-slate-200">AWS account name</Label>
            <Input
              value={accountName}
              readOnly
              className="border-slate-700 bg-slate-950/60 text-slate-300"
            />
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
            <p className="text-xs text-slate-500">Validate format first, then save to send `role_arn` to the backend account endpoint.</p>
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

          <div className="space-y-2">
            <Label className="text-slate-200">Implementation notes</Label>
            <Textarea
              value={notes}
              onChange={(event) => onNotesChange(event.target.value)}
              placeholder="Optional notes for reviewers, expected workspace, or permission boundaries."
              className="min-h-28 border-slate-700 bg-slate-900/40 text-white placeholder:text-slate-500 focus-visible:ring-teal-500"
              disabled={!isReadyForArn}
            />
          </div>

          <div className="flex flex-col gap-3 pt-2">
            <Button type="submit" className="w-full bg-teal-500 font-semibold text-slate-950 hover:bg-teal-400" disabled={!isReadyForArn || status === "verifying" || status === "saving"}>
              {status === "verifying" ? "Validating ARN..." : "Validate ARN format"}
              <ArrowRight className="h-4 w-4" />
            </Button>
            <Button
              type="button"
              variant="outline"
              className="w-full border-slate-700 bg-slate-900/40 text-slate-100 hover:bg-slate-800"
              onClick={() => void onSave()}
              disabled={!isReadyForArn || (status !== "verified" && status !== "saving")}
            >
              {status === "saving" ? "Submitting ARN..." : "Submit ARN and save account link"}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
