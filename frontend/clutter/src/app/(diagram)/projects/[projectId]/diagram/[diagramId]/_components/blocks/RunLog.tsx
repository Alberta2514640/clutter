"use client";

import CopyButton, { SMALL_ACTION } from "@/components/common/CopyButton";
import ErrorState from "@/components/common/ErrorState";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { adapterErrorMessage } from "@/lib/adapterApi";
import { useRunLogs } from "@/lib/features/deployments/hooks";
import { readableLog } from "@/lib/features/deployments/results";
import { failureAdvice, failureLines, runHeadline } from "@/lib/features/deployments/runLog";
import { RUN_ACTION, runLabel } from "@/lib/features/deployments/runState";
import { isActive, type Run } from "@/lib/features/deployments/types";
import { cn } from "@/lib/utils";
import { Download, Loader2, Maximize2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";

// RunHeadline is what a finished run's card leads with: what a deploy built
// ("Built 4 blocks (18 cloud resources)") or a destroy removed, counted from
// the run's own log, or what went wrong, in the log's own words, with advice
// for a known cloud error (runLog.failureAdvice), then B's failure line when
// it says something more. It reads the same log query as
// RunLog, so the log is fetched once.
export function RunHeadline({ token, organizationId, run }: { token: string | null; organizationId: string | null; run: Run }) {
  const logQ = useRunLogs(token, organizationId, run);
  const result = logQ.data ? runHeadline(logQ.data, run.status, run.action) : null;
  const failedLog = run.status === "failed" ? logQ.data : undefined;
  const more = failedLog ? failureLines(failedLog).slice(1) : [];
  const advice = failedLog ? failureAdvice(failedLog) : null;
  const failure = run.failure && run.failure !== result?.text ? run.failure : null;
  if (!result && !failure) return null;
  return (
    <div className="space-y-0.5">
      {result && (
        <p data-testid="run-result" className={cn("text-sm font-medium break-words", result.tone === "ok" ? "text-success" : "text-danger")}>
          {result.text}
        </p>
      )}
      {advice && (
        <p data-testid="failure-advice" className="text-foreground text-xs">
          {advice}
        </p>
      )}
      {more.map((l) => (
        <p key={l} className="text-danger text-xs break-words">
          {l}
        </p>
      ))}
      {failure && <p className="text-danger text-xs">{failure}</p>}
    </div>
  );
}

// LogText is a run's log in a scrolling box. It opens at its end and follows
// new lines only while the reader is at the bottom, so reading earlier lines
// never jumps.
function LogText({ text, className, testId }: { text: string; className?: string; testId?: string }) {
  const preRef = useRef<HTMLPreElement>(null);
  const following = useRef(true);

  useEffect(() => {
    const el = preRef.current;
    if (el && following.current) el.scrollTop = el.scrollHeight;
  }, [text]);

  return (
    <pre
      ref={preRef}
      data-testid={testId}
      tabIndex={0}
      aria-label="Run log"
      onScroll={(e) => {
        const el = e.currentTarget;
        following.current = el.scrollHeight - el.scrollTop - el.clientHeight < 24;
      }}
      className={cn(
        "bg-muted/60 text-foreground focus-visible:ring-ring/50 overflow-auto rounded-md border p-3 font-mono text-xs leading-relaxed whitespace-pre-wrap break-words outline-none focus-visible:ring-[3px]",
        className,
      )}>
      {text.trim() === "" ? "No output yet." : text}
    </pre>
  );
}

// A run's log, polled while the run is active, with ways to copy it, save it
// as a text file and read it in a large window. It shows B's machine lines
// in plain words (results.readableLog); a failed run's window leads with what
// went wrong. B removes credential values from it before it is stored.
export default function RunLog({
  token,
  organizationId,
  run,
  title,
  blockTitle,
  testId = "run-log",
}: {
  token: string | null;
  organizationId: string | null;
  run: Run;
  title?: string;
  // Names a block kind ("VM") in the log's "Built" lines.
  blockTitle?: (kind: string) => string;
  testId?: string;
}) {
  const logQ = useRunLogs(token, organizationId, run);
  const [expanded, setExpanded] = useState(false);
  const text = readableLog(logQ.data ?? "", blockTitle);
  const failed = run.status === "failed" ? failureLines(logQ.data ?? "") : [];
  const advice = run.status === "failed" ? failureAdvice(logQ.data ?? "") : null;
  const download = () => {
    const url = URL.createObjectURL(new Blob([text], { type: "text/plain;charset=utf-8" }));
    const a = document.createElement("a");
    a.href = url;
    a.download = `clutter-${RUN_ACTION[run.action].toLowerCase()}-${run.id}.txt`;
    a.click();
    URL.revokeObjectURL(url);
  };

  if (logQ.isLoading) {
    return (
      <div role="status" className="text-muted-foreground flex items-center gap-2 text-xs">
        <Loader2 className="size-3.5 animate-spin" aria-hidden />
        Loading the log…
      </div>
    );
  }
  if (logQ.isError) {
    return <ErrorState variant="inline" description={`Could not load the log. ${adapterErrorMessage(logQ.error)}`} onRetry={() => void logQ.refetch()} />;
  }
  const heading = title ?? runLabel(run);
  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between gap-2">
        <span className="text-muted-foreground flex items-center gap-1.5 text-xs">
          {isActive(run) && <Loader2 className="size-3 animate-spin" aria-hidden />}
          {isActive(run) ? "Live log" : "Log"}
        </span>
        <span className="flex items-center gap-1.5">
          {text.trim() !== "" && <CopyButton value={text} label="the log" />}
          {text.trim() !== "" && (
            <button type="button" className={SMALL_ACTION} onClick={download}>
              <Download className="size-3" aria-hidden />
              Download
            </button>
          )}
          <button type="button" className={SMALL_ACTION} onClick={() => setExpanded(true)}>
            <Maximize2 className="size-3" aria-hidden />
            Expand
          </button>
        </span>
      </div>
      <LogText text={text} testId={testId} className="max-h-72" />
      <Dialog open={expanded} onOpenChange={setExpanded}>
        <DialogContent className="flex h-[85dvh] max-w-[min(72rem,calc(100%-2rem))] flex-col sm:max-w-[min(72rem,calc(100%-2rem))]">
          <DialogHeader>
            <DialogTitle>{heading}</DialogTitle>
            <DialogDescription>
              {run.finished_at ? `Finished ${new Date(run.finished_at).toLocaleString()}` : isActive(run) ? "Running now. New lines appear as they arrive." : "Log"}
            </DialogDescription>
          </DialogHeader>
          {failed.length > 0 && (
            <div role="alert" className="border-danger/30 bg-danger/10 text-danger space-y-0.5 rounded-md border px-3 py-2 text-sm">
              {failed.map((l) => (
                <p key={l} className="break-words">
                  {l}
                </p>
              ))}
              {advice && <p className="text-foreground text-xs">{advice}</p>}
            </div>
          )}
          <LogText text={text} className="min-h-0 flex-1 text-[13px]" />
        </DialogContent>
      </Dialog>
    </div>
  );
}
