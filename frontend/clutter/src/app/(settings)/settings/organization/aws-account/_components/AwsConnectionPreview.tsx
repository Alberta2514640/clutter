import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import type { LinkStatus } from "./arn_page";

interface AwsConnectionPreviewProps {
  accountName: string;
  defaultRegion: string;
  launchUrl: string | null;
  linkedAccountId: string | null;
  status: LinkStatus;
}

export default function AwsConnectionPreview({ accountName, defaultRegion, launchUrl, linkedAccountId, status }: AwsConnectionPreviewProps) {
  return (
    <Card className="border-slate-800 bg-slate-950/40 shadow-none">
      <CardHeader>
        <CardTitle className="text-xl text-white">Connection preview</CardTitle>
        <CardDescription className="text-slate-400">This reflects local page state. A completed account only appears after the backend returns it in the account list.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-3 text-sm text-slate-300">
        <div className="flex items-center justify-between rounded-xl border border-slate-800 bg-slate-900/70 px-4 py-3">
          <span className="text-slate-400">Status</span>
          <span className="font-medium text-white">{status === "idle" ? "Waiting for input" : status.replace("-", " ")}</span>
        </div>
        <div className="flex items-center justify-between rounded-xl border border-slate-800 bg-slate-900/70 px-4 py-3">
          <span className="text-slate-400">AWS account name</span>
          <span className="font-medium text-white">{accountName.trim() || "Not set"}</span>
        </div>
        <div className="flex items-center justify-between rounded-xl border border-slate-800 bg-slate-900/70 px-4 py-3">
          <span className="text-slate-400">Region</span>
          <span className="font-medium text-white">{defaultRegion}</span>
        </div>
        <div className="flex items-center justify-between rounded-xl border border-slate-800 bg-slate-900/70 px-4 py-3">
          <span className="text-slate-400">Pending link ID</span>
          <span className="font-medium text-white">{linkedAccountId ?? "Not created yet"}</span>
        </div>
        <div className="flex items-center justify-between rounded-xl border border-slate-800 bg-slate-900/70 px-4 py-3">
          <span className="text-slate-400">Launch URL</span>
          <span className="max-w-[16rem] truncate font-medium text-white">{launchUrl ?? "Not generated yet"}</span>
        </div>
        <div className="rounded-xl border border-dashed border-slate-700 bg-slate-900/30 p-4 text-slate-400">
          Once your backend is ready, replace the simulated success state with the returned AWS account ID, trust check result, and permission diagnostics.
        </div>
      </CardContent>
    </Card>
  );
}
