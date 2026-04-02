"use client";

import { useEffect, useRef } from "react";
import { LiveLogsTabProps } from "./types";

export default function LiveLogsTab({
  taskArn,
  lines,
  isComplete,
  taskStatus,
  isLoading,
}: LiveLogsTabProps) {
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