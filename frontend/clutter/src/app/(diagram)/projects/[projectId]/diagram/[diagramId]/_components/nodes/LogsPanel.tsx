"use client";
import { useState } from "react";

export type LogEntry = {
  id: string;
  timestamp: Date;
  message: string;
};

type LogsPanelProps = {
  logs: LogEntry[];
};

export default function LogsPanel({ logs }: LogsPanelProps) {
  const [open, setOpen] = useState(false);

  return (
    <div className="absolute bottom-0 left-0 right-0 z-40">
      {/* Pull tab — right-aligned */}
      <div className="flex justify-end pr-6 cursor-pointer" onClick={() => setOpen((o) => !o)}>
        <div className="flex items-center gap-2.5 bg-neutral-900 border border-white/10 border-b-0 rounded-t-lg px-5 py-2.5 text-sm text-slate-400 hover:text-slate-200 select-none transition-colors">
          <svg
            width="14"
            height="14"
            viewBox="0 0 12 12"
            fill="none"
            className={`transition-transform duration-200 ${open ? "rotate-180" : ""}`}
          >
            <path d="M2 8L6 4L10 8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
          Deploy Logs
          {logs.length > 0 && (
            <span className="bg-white/10 rounded px-1.5 py-0.5 text-xs tabular-nums">
              {logs.length}
            </span>
          )}
        </div>
      </div>

      {/* Log panel */}
      <div
        className="overflow-hidden bg-neutral-900 border-t border-white/10 transition-all duration-300"
        style={{ height: open ? 260 : 0 }}
      >
        <div className="h-full overflow-y-auto p-4 font-mono text-xs text-slate-300 space-y-1">
          {logs.length === 0 ? (
            <p className="text-slate-600">No logs yet.</p>
          ) : (
            logs.map((entry) => (
              <div key={entry.id} className="flex gap-3">
                <span className="text-slate-600 shrink-0 tabular-nums">
                  {entry.timestamp.toLocaleTimeString("en-US", { hour12: false })}
                </span>
                <span>{entry.message}</span>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}