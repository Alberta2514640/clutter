"use client";

import { adapterErrorMessage } from "@/lib/adapterApi";
import { useRunLogs } from "@/lib/features/deployments/hooks";
import { runHeadline } from "@/lib/features/deployments/runLog";
import type { Run } from "@/lib/features/deployments/types";
import { Loader2 } from "lucide-react";
import { useEffect, useRef } from "react";

// RunHeadline is the one line a run's card leads with: what a finished
// deploy built ("Built 4 blocks (18 cloud resources)"), or a run's summary or
// first error. It reads the same log query as RunLog, so the log is fetched
// once. Nothing shows until there is a line to show.
export function RunHeadline({ token, organizationId, run, blocks }: { token: string | null; organizationId: string | null; run: Run; blocks: number }) {
  const logQ = useRunLogs(token, organizationId, run.id, run.status);
  const result = logQ.data ? runHeadline(logQ.data, run.status, run.action, blocks) : null;
  if (!result) return null;
  return (
    <p data-testid="run-result" className={`text-xs font-medium ${result.tone === "ok" ? "text-emerald-300" : "text-red-300"}`}>
      {result.text}
    </p>
  );
}

// A run's log, polled while the run is active. It opens scrolled to the end
// and follows new output unless the reader has scrolled up. B removes
// credential values from it before it is stored.
export default function RunLog({ token, organizationId, run }: { token: string | null; organizationId: string | null; run: Run }) {
  const logQ = useRunLogs(token, organizationId, run.id, run.status);
  const text = logQ.data ?? "";
  const preRef = useRef<HTMLPreElement>(null);
  const following = useRef(true);

  useEffect(() => {
    const el = preRef.current;
    if (el && following.current) el.scrollTop = el.scrollHeight;
  }, [text]);

  if (logQ.isLoading) {
    return (
      <div role="status" className="flex items-center gap-2 text-xs text-slate-400">
        <Loader2 className="h-3.5 w-3.5 animate-spin" />
        Loading the log…
      </div>
    );
  }
  if (logQ.isError) return <p className="text-xs text-red-400">Could not load the log: {adapterErrorMessage(logQ.error)}</p>;
  return (
    <div className="space-y-1">
      <pre
        ref={preRef}
        data-testid="run-log"
        tabIndex={0}
        aria-label="Run log"
        onScroll={(e) => {
          const el = e.currentTarget;
          following.current = el.scrollHeight - el.scrollTop - el.clientHeight < 24;
        }}
        className="max-h-64 overflow-auto whitespace-pre rounded-md border border-slate-800 bg-black/60 p-2 font-mono text-[11px] leading-relaxed text-slate-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-teal-300">
        {text.trim() === "" ? "No output yet." : text}
      </pre>
    </div>
  );
}
