"use client";

import { useState } from "react";
import { AlertCircle, ArrowRight, CheckCircle2, Copy, ExternalLink, KeyRound, ShieldCheck, Sparkles } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";

type LinkStatus = "idle" | "awaiting-role-arn" | "verifying" | "verified" | "saving" | "error";

const ROLE_ARN_PATTERN = /^arn:aws:iam::\d{12}:role\/[\w+=,.@\-_/]+$/;
const externalId = "clutter-demo-external-id";
const clutterAccountId = "123456789012";
const templateLaunchUrl = "https://console.aws.amazon.com/cloudformation/home";

export default function IamRole() {
  const [accountAlias, setAccountAlias] = useState("");
  const [roleArn, setRoleArn] = useState("");
  const [defaultRegion, setDefaultRegion] = useState("us-west-2");
  const [notes, setNotes] = useState("");
  const [status, setStatus] = useState<LinkStatus>("idle");
  const [error, setError] = useState("");
  const [copiedField, setCopiedField] = useState<"externalId" | "trustPolicy" | null>(null);

  const arnLooksValid = ROLE_ARN_PATTERN.test(roleArn.trim());

  const trustPolicy = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::${clutterAccountId}:root"
      },
      "Action": "sts:AssumeRole",
      "Condition": {
        "StringEquals": {
          "sts:ExternalId": "${externalId}"
        }
      }
    }
  ]
}`;

  const handleCopy = async (value: string, field: "externalId" | "trustPolicy") => {
    try {
      await navigator.clipboard.writeText(value);
      setCopiedField(field);
      window.setTimeout(() => setCopiedField(null), 1500);
    } catch {
      setCopiedField(null);
    }
  };

  const handleVerify = async () => {
    if (!accountAlias.trim()) {
      setStatus("error");
      setError("Add an account alias before verifying the role.");
      return;
    }

    if (!arnLooksValid) {
      setStatus("error");
      setError("Enter a valid IAM Role ARN in the format arn:aws:iam::<account-id>:role/<role-name>.");
      return;
    }

    setError("");
    setStatus("verifying");

    window.setTimeout(() => {
      setStatus("verified");
    }, 900);
  };

  const handleSave = async () => {
    if (status !== "verified") {
      setStatus("error");
      setError("Verify the role before saving the account link.");
      return;
    }

    setError("");
    setStatus("saving");

    window.setTimeout(() => {
      setStatus("verified");
    }, 700);
  };

  return (
    <div className="space-y-8 text-white">
      <Card className="border-slate-800/50 bg-slate-900/50 shadow-2xl backdrop-blur-xl">
        <CardHeader className="gap-4 border-b border-slate-800/60">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
            <div className="flex items-start gap-4">
              <div className="rounded-2xl border border-teal-500/20 bg-teal-500/10 p-3">
                <KeyRound className="h-7 w-7 text-teal-400" />
              </div>

              <div className="space-y-2">
                <CardTitle className="flex items-center gap-2 text-3xl text-white">
                  Connect an AWS Account
                  <Sparkles className="h-5 w-5 text-teal-400" />
                </CardTitle>
                <CardDescription className="max-w-3xl text-sm leading-6 text-slate-400">
                  Clutter deploys infrastructure into your AWS account by assuming a secure IAM role. Create the role in AWS, paste the Role ARN here, then verify and save the connection.
                </CardDescription>
              </div>
            </div>

            <div className="grid min-w-[220px] gap-3 rounded-2xl border border-slate-700 bg-slate-800/40 p-4 text-sm">
              <div className="flex items-center gap-2 text-slate-200">
                <ShieldCheck className="h-4 w-4 text-teal-400" />
                No long-lived AWS keys
              </div>
              <div className="text-slate-400">Frontend only stores metadata like alias, role ARN, and region.</div>
            </div>
          </div>
        </CardHeader>

        <CardContent className="grid gap-6 px-6 py-6 lg:grid-cols-[1.1fr_0.9fr]">
          <section className="space-y-6">
            {status === "error" && error && (
              <Alert className="border-red-500/30 bg-red-500/10 text-red-100">
                <AlertCircle className="h-4 w-4" />
                <AlertTitle>Verification blocked</AlertTitle>
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}

            {status === "verified" && (
              <Alert className="border-teal-500/30 bg-teal-500/10 text-teal-50">
                <CheckCircle2 className="h-4 w-4 text-teal-300" />
                <AlertTitle>Connection verified</AlertTitle>
                <AlertDescription>Role trust looks correct. Replace the simulated verify handler with your backend STS check next.</AlertDescription>
              </Alert>
            )}

            <Card className="border-slate-800 bg-slate-950/40 shadow-none">
              <CardHeader>
                <CardTitle className="text-xl text-white">1. Why Clutter needs this role</CardTitle>
                <CardDescription className="text-slate-400">This should explain the deployment boundary before the user leaves for AWS.</CardDescription>
              </CardHeader>
              <CardContent className="grid gap-3 text-sm text-slate-300">
                <div className="rounded-xl border border-slate-800 bg-slate-900/70 p-4">Clutter assumes a cross-account IAM role only when running deployment actions for this workspace.</div>
                <div className="rounded-xl border border-slate-800 bg-slate-900/70 p-4">No AWS access key IDs, secret keys, or session tokens are collected in the browser.</div>
                <div className="rounded-xl border border-slate-800 bg-slate-900/70 p-4">You can keep permissions constrained to the resources your deployment engine actually manages.</div>
              </CardContent>
            </Card>

            <Card className="border-slate-800 bg-slate-950/40 shadow-none">
              <CardHeader>
                <CardTitle className="text-xl text-white">2. Create the IAM role in AWS</CardTitle>
                <CardDescription className="text-slate-400">Prefer CloudFormation for the first version. Keep the manual instructions visible for advanced users.</CardDescription>
              </CardHeader>
              <CardContent className="space-y-5">
                <div className="flex flex-col gap-3 sm:flex-row">
                  <Button type="button" className="bg-teal-500 font-semibold text-slate-950 hover:bg-teal-400" onClick={() => window.open(templateLaunchUrl, "_blank", "noopener,noreferrer")}>
                    Open AWS Console to Create Role
                    <ExternalLink className="h-4 w-4" />
                  </Button>
                  <Button type="button" variant="outline" className="border-slate-700 bg-slate-900/40 text-slate-200 hover:bg-slate-800" onClick={() => setStatus("awaiting-role-arn")}>
                    I&apos;ll create the role manually
                  </Button>
                </div>

                <div className="grid gap-3 rounded-2xl border border-slate-800 bg-slate-900/60 p-4 text-sm text-slate-300">
                  <div className="flex items-center justify-between gap-3 rounded-xl border border-slate-700/70 bg-slate-950/40 px-4 py-3">
                    <div>
                      <p className="text-xs uppercase tracking-[0.2em] text-slate-500">External ID</p>
                      <p className="font-mono text-sm text-slate-100">{externalId}</p>
                    </div>
                    <Button type="button" variant="ghost" size="sm" className="text-slate-300 hover:bg-slate-800 hover:text-white" onClick={() => handleCopy(externalId, "externalId")}>
                      <Copy className="h-4 w-4" />
                      {copiedField === "externalId" ? "Copied" : "Copy"}
                    </Button>
                  </div>

                  <div className="grid gap-2 text-slate-400">
                    <p>1. Open the AWS Console and create a stack or role in the target account.</p>
                    <p>2. Configure the trust relationship to allow Clutter&apos;s AWS account to assume the role.</p>
                    <p>3. Copy the created Role ARN from AWS, then paste it into the verification form below.</p>
                  </div>

                  <div className="rounded-xl border border-slate-700/70 bg-slate-950/40 p-4">
                    <div className="mb-3 flex items-center justify-between gap-3">
                      <div>
                        <p className="text-sm font-semibold text-white">Trust policy reference</p>
                        <p className="text-xs text-slate-500">Use this as the baseline for your manual IAM role setup.</p>
                      </div>
                      <Button type="button" variant="ghost" size="sm" className="text-slate-300 hover:bg-slate-800 hover:text-white" onClick={() => handleCopy(trustPolicy, "trustPolicy")}>
                        <Copy className="h-4 w-4" />
                        {copiedField === "trustPolicy" ? "Copied" : "Copy JSON"}
                      </Button>
                    </div>
                    <Textarea value={trustPolicy} readOnly className="min-h-44 border-slate-700 bg-slate-950 text-xs text-slate-300" />
                  </div>
                </div>
              </CardContent>
            </Card>
          </section>

          <section className="space-y-6">
            <Card className="border-slate-800 bg-slate-950/40 shadow-none">
              <CardHeader>
                <CardTitle className="text-xl text-white">3. Paste role details and verify</CardTitle>
                <CardDescription className="text-slate-400">This is the form you&apos;ll eventually connect to `verifyAccountLink` and `createAccountLink`.</CardDescription>
              </CardHeader>
              <CardContent>
                <form
                  className="space-y-5"
                  onSubmit={(event) => {
                    event.preventDefault();
                    void handleVerify();
                  }}>
                  <div className="space-y-2">
                    <Label className="text-slate-200">Account alias</Label>
                    <Input
                      value={accountAlias}
                      onChange={(event) => setAccountAlias(event.target.value)}
                      placeholder="e.g. Production AWS"
                      className="border-slate-700 bg-slate-900/40 text-white placeholder:text-slate-500 focus-visible:ring-teal-500"
                    />
                    <p className="text-xs text-slate-500">Use a label your team will recognize when attaching this account to a workspace.</p>
                  </div>

                  <div className="space-y-2">
                    <Label className="text-slate-200">Role ARN</Label>
                    <Input
                      value={roleArn}
                      onChange={(event) => setRoleArn(event.target.value)}
                      placeholder="arn:aws:iam::123456789012:role/ClutterDeployRole"
                      className="border-slate-700 bg-slate-900/40 font-mono text-white placeholder:text-slate-500 focus-visible:ring-teal-500"
                    />
                    <p className="text-xs text-slate-500">Client-side validation only checks format. Real trust validation must happen in the backend.</p>
                  </div>

                  <div className="space-y-2">
                    <Label className="text-slate-200">Default region</Label>
                    <Select value={defaultRegion} onValueChange={setDefaultRegion}>
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
                      onChange={(event) => setNotes(event.target.value)}
                      placeholder="Optional notes for reviewers, expected workspace, or permission boundaries."
                      className="min-h-28 border-slate-700 bg-slate-900/40 text-white placeholder:text-slate-500 focus-visible:ring-teal-500"
                    />
                  </div>

                  <div className="flex flex-col gap-3 pt-2">
                    <Button type="submit" className="w-full bg-teal-500 font-semibold text-slate-950 hover:bg-teal-400" disabled={status === "verifying" || status === "saving"}>
                      {status === "verifying" ? "Verifying role..." : "Verify connection"}
                      <ArrowRight className="h-4 w-4" />
                    </Button>
                    <Button type="button" variant="outline" className="w-full border-slate-700 bg-slate-900/40 text-slate-100 hover:bg-slate-800" onClick={() => void handleSave()} disabled={status !== "verified" && status !== "saving"}>
                      {status === "saving" ? "Saving account link..." : "Save account link"}
                    </Button>
                  </div>
                </form>
              </CardContent>
            </Card>

            <Card className="border-slate-800 bg-slate-950/40 shadow-none">
              <CardHeader>
                <CardTitle className="text-xl text-white">Connection preview</CardTitle>
                <CardDescription className="text-slate-400">Use this card later for backend verification results and workspace attachment.</CardDescription>
              </CardHeader>
              <CardContent className="space-y-3 text-sm text-slate-300">
                <div className="flex items-center justify-between rounded-xl border border-slate-800 bg-slate-900/70 px-4 py-3">
                  <span className="text-slate-400">Status</span>
                  <span className="font-medium text-white">{status === "idle" ? "Waiting for input" : status.replace("-", " ")}</span>
                </div>
                <div className="flex items-center justify-between rounded-xl border border-slate-800 bg-slate-900/70 px-4 py-3">
                  <span className="text-slate-400">Account alias</span>
                  <span className="font-medium text-white">{accountAlias.trim() || "Not set"}</span>
                </div>
                <div className="flex items-center justify-between rounded-xl border border-slate-800 bg-slate-900/70 px-4 py-3">
                  <span className="text-slate-400">Region</span>
                  <span className="font-medium text-white">{defaultRegion}</span>
                </div>
                <div className="rounded-xl border border-dashed border-slate-700 bg-slate-900/30 p-4 text-slate-400">
                  Once your backend is ready, replace the simulated success state with the returned AWS account ID, trust check result, and permission diagnostics.
                </div>
              </CardContent>
            </Card>
          </section>
        </CardContent>
      </Card>
    </div>
  );
}
