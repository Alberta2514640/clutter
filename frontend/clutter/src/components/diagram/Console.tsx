"use client";

import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";

/**
 * Console event bus (global)
 * --------------------------
 * Fire logs from anywhere in your app:
 *
 *   import { logInfo, logWarn, logError, logSuccess, log } from "@/components/diagram/Console";
 *   logInfo("Loaded canvas", { canvasId });
 *   logWarn("Edge rejected: rule violation");
 *   logError("Failed to save", { status: 409 });
 *   logSuccess("Plan finished", { runId });
 *   log("debug", "Raw payload", payload); // custom level
 */

export type LogLevel = "debug" | "info" | "success" | "warn" | "error";

export type ConsoleEvent = {
  id: string;
  ts: number; // epoch ms
  level: LogLevel;
  message: string;
  meta?: unknown;
};

const BUS_EVENT = "clutter:console";

export function log(level: LogLevel, message: string, meta?: unknown) {
  const detail: ConsoleEvent = {
    id: `${Date.now()}-${Math.random().toString(36).slice(2)}`,
    ts: Date.now(),
    level,
    message,
    meta,
  };
  window.dispatchEvent(new CustomEvent(BUS_EVENT, { detail }));
}
export const logDebug = (m: string, meta?: unknown) => log("debug", m, meta);
export const logInfo = (m: string, meta?: unknown) => log("info", m, meta);
export const logSuccess = (m: string, meta?: unknown) => log("success", m, meta);
export const logWarn = (m: string, meta?: unknown) => log("warn", m, meta);
export const logError = (m: string, meta?: unknown) => log("error", m, meta);

/**
 * Console UI
 * ----------
 * - Filter by level
 * - Text search
 * - Clear / Copy
 * - Auto-scroll
 */
export default function Console() {
  const [items, setItems] = useState<ConsoleEvent[]>([]);
  const [levels, setLevels] = useState<Record<LogLevel, boolean>>({
    debug: false, // default hidden (noisy)
    info: true,
    success: true,
    warn: true,
    error: true,
  });
  const [query, setQuery] = useState("");
  const [autoScroll, setAutoScroll] = useState(true);

  const endRef = useRef<HTMLDivElement | null>(null);

  // Subscribe to global console events
  useEffect(() => {
    const onEvent = (ev: Event) => {
      const e = ev as CustomEvent<ConsoleEvent>;
      setItems((prev) => [...prev, e.detail]);
    };
    window.addEventListener(BUS_EVENT, onEvent);
    return () => window.removeEventListener(BUS_EVENT, onEvent);
  }, []);

  // Auto-scroll to bottom
  useEffect(() => {
    if (!autoScroll) return;
    endRef.current?.scrollIntoView({ behavior: "smooth", block: "end" });
  }, [items, autoScroll]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return items.filter((it) => {
      if (!levels[it.level]) return false;
      if (!q) return true;
      const hay = (it.message?.toLowerCase?.() ?? "") + " " + (safeJSON(it.meta)?.toLowerCase?.() ?? "");
      return hay.includes(q);
    });
  }, [items, levels, query]);

  const onToggleLevel = useCallback((lvl: LogLevel) => setLevels((s) => ({ ...s, [lvl]: !s[lvl] })), []);

  const onClear = useCallback(() => setItems([]), []);
  const onCopy = useCallback(async () => {
    const text = filtered.map((i) => `${fmtTime(i.ts)} [${i.level.toUpperCase()}] ${i.message}${i.meta ? `\n  ${indent(safeJSON(i.meta), 2)}` : ""}`).join("\n");
    await navigator.clipboard.writeText(text);
  }, [filtered]);

  return (
    <div className="flex flex-col h-80 w-full border rounded-lg overflow-hidden bg-black/95 text-neutral-100">
      {/* Header / controls */}
      <div className="flex flex-wrap items-center gap-2 border-b border-neutral-800 px-3 py-2">
        <span className="text-sm font-medium text-neutral-300">Console</span>
        <div className="ml-auto flex items-center gap-2">
          <LevelPill label="ERR" active={levels.error} color="text-red-300 border-red-500" onClick={() => onToggleLevel("error")} />
          <LevelPill label="WARN" active={levels.warn} color="text-amber-300 border-amber-500" onClick={() => onToggleLevel("warn")} />
          <LevelPill label="OK" active={levels.success} color="text-emerald-300 border-emerald-500" onClick={() => onToggleLevel("success")} />
          <LevelPill label="INFO" active={levels.info} color="text-sky-300 border-sky-500" onClick={() => onToggleLevel("info")} />
          <LevelPill label="DBG" active={levels.debug} color="text-neutral-300 border-neutral-500" onClick={() => onToggleLevel("debug")} />

          <input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Search..." className="bg-neutral-900 text-sm px-2 py-1 rounded border border-neutral-700 focus:outline-none focus:ring-2 focus:ring-sky-500" />

          <button onClick={() => setAutoScroll((v) => !v)} className={`text-xs px-2 py-1 rounded border ${autoScroll ? "border-emerald-500 text-emerald-300" : "border-neutral-600 text-neutral-300"}`} title="Toggle auto-scroll">
            {autoScroll ? "Auto-scroll" : "Manual"}
          </button>

          <button onClick={onCopy} className="text-xs px-2 py-1 rounded border border-neutral-600 hover:border-neutral-400" title="Copy visible logs">
            Copy
          </button>
          <button onClick={onClear} className="text-xs px-2 py-1 rounded border border-red-600 text-red-300 hover:border-red-400" title="Clear all logs">
            Clear
          </button>
        </div>
      </div>

      {/* Log list */}
      <div className="flex-1 overflow-auto font-mono text-[12px] leading-5 px-3 py-2">
        {filtered.length === 0 ? (
          <div className="text-neutral-400/70">No logs yet.</div>
        ) : (
          filtered.map((i) => (
            <div key={i.id} className="whitespace-pre-wrap">
              <span className="text-neutral-500">{fmtTime(i.ts)}</span> <LevelBadge level={i.level} /> <span className="text-neutral-100">{i.message}</span>
              {i.meta ? (
                <>
                  {"\n"}
                  <span className="text-neutral-400">{indent(safeJSON(i.meta), 2)}</span>
                </>
              ) : null}
            </div>
          ))
        )}
        <div ref={endRef} />
      </div>
    </div>
  );
}

/* ---------------- helpers / subcomponents ---------------- */

function LevelBadge({ level }: { level: LogLevel }) {
  const cls = {
    error: "text-red-300",
    warn: "text-amber-300",
    success: "text-emerald-300",
    info: "text-sky-300",
    debug: "text-neutral-300",
  }[level];
  return <span className={cls + " font-semibold"}>[{level.toUpperCase()}]</span>;
}

function LevelPill({
  label,
  active,
  color,
  onClick,
}: {
  label: string;
  active: boolean;
  color: string; // e.g., "text-red-300 border-red-500"
  onClick: () => void;
}) {
  return (
    <button onClick={onClick} className={`text-[11px] px-2 py-0.5 rounded-full border ${color} ${active ? "opacity-100" : "opacity-40"} hover:opacity-100`} title={`Toggle ${label}`}>
      {label}
    </button>
  );
}

function fmtTime(ts: number) {
  const d = new Date(ts);
  const hh = String(d.getHours()).padStart(2, "0");
  const mm = String(d.getMinutes()).padStart(2, "0");
  const ss = String(d.getSeconds()).padStart(2, "0");
  const ms = String(d.getMilliseconds()).padStart(3, "0");
  return `${hh}:${mm}:${ss}.${ms}`;
}

function safeJSON(v: unknown) {
  try {
    return JSON.stringify(v, null, 2);
  } catch {
    return String(v);
  }
}

function indent(s: string, n = 2) {
  const pad = " ".repeat(n);
  return s
    .split("\n")
    .map((l) => (l.length ? pad + l : l))
    .join("\n");
}
