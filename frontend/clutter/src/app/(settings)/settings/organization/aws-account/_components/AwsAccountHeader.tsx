import { ExternalLink, KeyRound, ShieldCheck } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

const AWS_SETUP_GUIDE_URL = "/guide/AWS_Account_Link_Steps.pdf";

export default function AwsAccountHeader({ hasLinkedAccount }: { hasLinkedAccount: boolean }) {
  return (
    <Card className="border-slate-800/50 bg-slate-900/50 shadow-2xl backdrop-blur-xl">
      <CardHeader className="gap-6">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="flex items-start gap-4">
            <div className="rounded-2xl border border-teal-500/20 bg-teal-500/10 p-3">
              <KeyRound className="h-7 w-7 text-teal-400" aria-hidden="true" />
            </div>

            <div className="space-y-2">
              <CardTitle className="text-3xl text-white">{hasLinkedAccount ? "Link another AWS account" : "Link an AWS account"}</CardTitle>
              <CardDescription className="max-w-3xl text-sm leading-6 text-slate-400">
                Clutter builds your diagrams in your AWS account through a role you create there. It takes two steps: open AWS to create the role, then paste the role&apos;s ARN here.
              </CardDescription>
            </div>
          </div>

          <div className="grid min-w-[220px] gap-3 rounded-2xl border border-slate-700 bg-slate-800/40 p-4 text-sm">
            <div className="flex items-center gap-2 text-slate-200">
              <ShieldCheck className="h-4 w-4 text-teal-400" aria-hidden="true" />
              No AWS keys needed
            </div>
            <div className="text-slate-400">Clutter never asks for your AWS access keys. You can delete the role in AWS at any time.</div>
          </div>
        </div>

        <div className="flex flex-col gap-3 rounded-2xl border border-slate-700 bg-slate-950/50 px-5 py-4 text-sm text-slate-300 sm:flex-row sm:items-center sm:justify-between">
          <span className="max-w-3xl text-lg leading-8 text-slate-200">Want to see each step first? Open the guide with screenshots.</span>
          <Button variant="ghost" size="sm" asChild className="shrink-0 text-base text-slate-100 hover:bg-slate-800 hover:text-white">
            <a href={AWS_SETUP_GUIDE_URL} target="_blank" rel="noreferrer">
              Open the setup guide
              <ExternalLink className="h-4 w-4" aria-hidden="true" />
            </a>
          </Button>
        </div>
      </CardHeader>
    </Card>
  );
}
