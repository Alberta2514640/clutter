import { ArrowRight } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { awsRoleArnExample } from "@/lib/features/organization/rules";
import type { LinkStatus } from "./AwsLinkFlow";

interface AwsRoleVerificationFormProps {
  accountName: string;
  isReadyForArn: boolean;
  roleArn: string;
  status: LinkStatus;
  onRoleArnChange: (value: string) => void;
  onSave: () => Promise<void>;
}

export default function AwsRoleVerificationForm({ accountName, isReadyForArn, roleArn, status, onRoleArnChange, onSave }: AwsRoleVerificationFormProps) {
  return (
    <Card className="h-full border-slate-800 bg-slate-950/40 shadow-none">
      <CardHeader>
        <CardTitle className="text-xl text-white">2. Paste the role ARN</CardTitle>
        <CardDescription className="text-slate-400">
          {isReadyForArn ? "Paste the role ARN from the stack's Outputs tab, then link the account." : "Finish step 1 first. This step opens when AWS is ready."}
        </CardDescription>
      </CardHeader>
      <CardContent className="flex-1">
        <form
          className="space-y-5"
          onSubmit={(event) => {
            event.preventDefault();
            void onSave();
          }}>
          <div className="space-y-2">
            <Label htmlFor="aws-link-account-name" className="text-slate-200">
              Account name
            </Label>
            <Input id="aws-link-account-name" value={accountName} readOnly className="border-slate-700 bg-slate-950/60 text-slate-300" />
          </div>

          <div className="space-y-2">
            <Label htmlFor="aws-role-arn" className="text-slate-200">
              Role ARN
            </Label>
            <Input
              id="aws-role-arn"
              value={roleArn}
              onChange={(event) => onRoleArnChange(event.target.value)}
              placeholder={awsRoleArnExample}
              aria-describedby="aws-role-arn-hint"
              className="border-slate-700 bg-slate-900/40 font-mono text-white placeholder:text-slate-500 focus-visible:ring-teal-500"
              disabled={!isReadyForArn}
            />
            <p id="aws-role-arn-hint" className="text-xs text-slate-500">
              It starts with arn:aws:iam:: and ends with AllowClutterToDeployTerraformRole and a code of 8 letters and digits.
            </p>
          </div>

          <div className="flex flex-col gap-3 pt-2">
            <Button type="submit" className="w-full bg-teal-500 font-semibold text-slate-950 hover:bg-teal-400" disabled={!isReadyForArn || status === "saving"}>
              {status === "saving" ? "Linking the account..." : "Link account"}
              <ArrowRight className="h-4 w-4" aria-hidden="true" />
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
