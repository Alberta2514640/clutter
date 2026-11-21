"use client";

import { cn } from "@/lib/utils";
import { Check, Copy } from "lucide-react";
import { useEffect, useState } from "react";

// A small bordered action beside a value: Copy here, and Open beside a
// result's address in B's deploy panel.
export const SMALL_ACTION =
  "bg-background hover:border-primary/50 hover:bg-accent focus-visible:ring-ring/50 inline-flex h-6 shrink-0 items-center gap-1 rounded-md border px-2 text-[11px] outline-none transition-colors focus-visible:ring-[3px]";

// CopyButton puts value on the clipboard and says so for a moment. A browser
// that refuses the clipboard gets a plain sentence; the value stays on screen
// to select by hand. label names the value ("Copy team ID").
export default function CopyButton({ value, label, className }: { value: string; label: string; className?: string }) {
  const [state, setState] = useState<"idle" | "copied" | "failed">("idle");
  useEffect(() => {
    if (state === "idle") return;
    const t = window.setTimeout(() => setState("idle"), 2000);
    return () => window.clearTimeout(t);
  }, [state]);
  return (
    <button
      type="button"
      aria-label={`Copy ${label}`}
      onClick={() => {
        if (!navigator.clipboard) return setState("failed");
        navigator.clipboard.writeText(value).then(
          () => setState("copied"),
          () => setState("failed"),
        );
      }}
      className={cn(SMALL_ACTION, className)}>
      {state === "copied" ? <Check className="size-3" aria-hidden /> : <Copy className="size-3" aria-hidden />}
      <span aria-live="polite">{state === "copied" ? "Copied" : state === "failed" ? "Select it to copy" : "Copy"}</span>
    </button>
  );
}
