"use client";

import { useLogFileUrl, useLogFiles } from "@/lib/features/logs/hooks";
import type { LiveLogLine } from "@/lib/features/logs/types";
import { useEffect, useRef, useState } from "react";

// ─── Types ────────────────────────────────────────────────────────────────────

type Tab = "live" | "files";

type LogsPanelProps = {
  token: string | null;
  orgId: string | null;
  projectId: string;
  diagramId: string;
  taskArn: string | null;
  liveLogs: {
    lines: LiveLogLine[];
    isComplete: boolean;
    taskStatus: string;
    isLoading: boolean;
    isError: boolean;
    error: unknown;
  };
};

// ─── Live tab ─────────────────────────────────────────────────────────────────

function LiveLogsTab({
  taskArn,
  lines,
  isComplete,
  taskStatus,
  isLoading,
}: {
  taskArn: string | null;
  lines: LiveLogLine[];
  isComplete: boolean;
  taskStatus: string;
  isLoading: boolean;
}) {
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [lines.length]);

  if (!taskArn) {
    return (
      <p className="text-slate-600 text-xs font-mono p-4">
        No active deployment. Run Apply or Destroy to see live logs.
      </p>
    );
  }

  if (isLoading && lines.length === 0) {
    return (
      <div className="flex items-center gap-2 p-4 text-xs text-slate-500 font-mono">
        <span className="h-2 w-2 rounded-full bg-amber-400 animate-pulse" />
        Waiting for task to start…
      </div>
    );
  }

  return (
    <div className="h-full overflow-y-auto p-4 font-mono text-xs text-slate-300 space-y-0.5">
      <div className="flex items-center gap-2 mb-3">
        {isComplete ? (
          <span className="flex items-center gap-1.5 text-emerald-400">
            <span className="h-2 w-2 rounded-full bg-emerald-400" />
            {taskStatus || "Complete"}
          </span>
        ) : (
          <span className="flex items-center gap-1.5 text-amber-400">
            <span className="h-2 w-2 rounded-full bg-amber-400 animate-pulse" />
            {taskStatus || "Running…"}
          </span>
        )}
      </div>

      {lines.map((line, i) => {
        const display = line.message.split("\r").pop() ?? line.message;
        return (
          <div key={i} className="flex gap-3 leading-5">
            <span className="text-slate-600 shrink-0 tabular-nums w-[72px]">
              {new Date(line.timestamp).toLocaleTimeString("en-US", { hour12: false })}
            </span>
            <span className="whitespace-pre-wrap break-all">{display}</span>
          </div>
        );
      })}

      <div ref={bottomRef} />
    </div>
  );
}

// ─── Files tab ────────────────────────────────────────────────────────────────

function FilesTab({ token, orgId, projectId, diagramId }: Omit<LogsPanelProps, "taskArn" | "liveLogs">) {
  const { data: files, isLoading, isError } = useLogFiles(token, orgId, projectId, diagramId);
  const [downloading, setDownloading] = useState<string | null>(null);

  if (isLoading) return <p className="text-slate-500 text-xs font-mono p-4">Loading log files…</p>;
  if (isError) return <p className="text-red-400 text-xs font-mono p-4">Failed to load log files.</p>;
  if (!files?.length) return <p className="text-slate-600 text-xs font-mono p-4">No log files found.</p>;

  return (
    <div className="h-full overflow-y-auto p-4 space-y-2">
      {files.map((f) => (
        <FileRow
          key={`${f.deploymentId}-${f.file}`}
          token={token}
          orgId={orgId}
          projectId={projectId}
          diagramId={diagramId}
          deploymentId={f.deploymentId}
          file={f.file}
          downloading={downloading}
          setDownloading={setDownloading}
        />
      ))}
    </div>
  );
}

function FileRow({
  token,
  orgId,
  projectId,
  diagramId,
  deploymentId,
  file,
  downloading,
  setDownloading,
}: {
  token: string | null;
  orgId: string | null;
  projectId: string;
  diagramId: string;
  deploymentId: string;
  file: string;
  downloading: string | null;
  setDownloading: (k: string | null) => void;
}) {
  const key = `${deploymentId}-${file}`;
  const { refetch } = useLogFileUrl(token, orgId, projectId, diagramId, deploymentId, file);

  const handleView = async () => {
    setDownloading(key);
    try {
      const { data: url } = await refetch();
      if (url) window.open(url, "_blank");
    } finally {
      setDownloading(null);
    }
  };

  const isDownloading = downloading === key;

  return (
    <div className="flex items-center justify-between rounded-md bg-white/5 border border-white/10 px-3 py-2 text-xs">
      <div className="flex flex-col gap-0.5 min-w-0">
        <span className="text-slate-200 font-mono truncate">{file}</span>
        <span className="text-slate-500 tabular-nums">{deploymentId}</span>
      </div>
      <button
        onClick={handleView}
        disabled={isDownloading}
        className="ml-4 shrink-0 flex items-center gap-1.5 rounded px-2.5 py-1 text-xs bg-white/10 hover:bg-white/20 text-slate-300 hover:text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {isDownloading ? (
          <>
            <span className="h-3 w-3 rounded-full border border-slate-400 border-t-transparent animate-spin" />
            Opening…
          </>
        ) : (
          <>
            <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
              <path
                d="M6 1v7M3 5l3 3 3-3M2 10h8"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
            View
          </>
        )}
      </button>
    </div>
  );
}

// ─── Main panel ───────────────────────────────────────────────────────────────

export default function LogsPanel({
  token,
  orgId,
  projectId,
  diagramId,
  taskArn,
  liveLogs,
}: LogsPanelProps) {
  const [open, setOpen] = useState(() => !!taskArn);
  const [tab, setTab] = useState<Tab>("live");

  const { lines, isComplete, taskStatus, isLoading } = liveLogs;

  return (
    <div className="absolute bottom-0 left-0 right-0 z-40">
      <div className="flex justify-end pr-6">
        <button
          onClick={() => setOpen((o) => !o)}
          className="flex items-center gap-2.5 bg-neutral-900 border border-white/10 border-b-0 rounded-t-lg px-5 py-2.5 text-sm text-slate-400 hover:text-slate-200 select-none transition-colors cursor-pointer"
        >
          <svg
            width="14"
            height="14"
            viewBox="0 0 12 12"
            fill="none"
            className={`transition-transform duration-200 ${open ? "rotate-180" : ""}`}
          >
            <path
              d="M2 8L6 4L10 8"
              stroke="currentColor"
              strokeWidth="1.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
          Deploy Logs
          {taskArn &&
            (isComplete ? (
              <span className="flex items-center gap-1 text-emerald-400 text-xs">
                <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
                Complete
              </span>
            ) : (
              <span className="flex items-center gap-1 text-amber-400 text-xs">
                <span className="h-1.5 w-1.5 rounded-full bg-amber-400 animate-pulse" />
                Live
              </span>
            ))}
        </button>
      </div>

      <div
        className="overflow-hidden bg-neutral-900 border-t border-white/10 transition-all duration-300"
        style={{ height: open ? 280 : 0 }}
      >
        <div className="flex items-center border-b border-white/10 px-4 gap-1 h-9 shrink-0">
          {(["live", "files"] as Tab[]).map((t) => (
            <button
              key={t}
              onClick={() => setTab(t)}
              className={`px-3 py-1 rounded text-xs capitalize transition-colors ${
                tab === t ? "bg-white/10 text-slate-100" : "text-slate-500 hover:text-slate-300"
              }`}
            >
              {t === "live" ? "Live" : "Log Files"}
            </button>
          ))}
        </div>

        <div className="h-[calc(100%-36px)]">
          {tab === "live" ? (
            <LiveLogsTab
              taskArn={taskArn}
              lines={lines}
              isComplete={isComplete}
              taskStatus={taskStatus}
              isLoading={isLoading}
            />
          ) : (
            <FilesTab token={token} orgId={orgId} projectId={projectId} diagramId={diagramId} />
          )}
        </div>
      </div>
    </div>
  );
}