import { Loader2 } from "lucide-react";

// The Connections page's loading line and its boxed messages.

export function Loading({ text }: { text: string }) {
  return (
    <div role="status" className="flex items-center gap-2 px-1 py-3 text-sm text-slate-400">
      <Loader2 className="h-4 w-4 animate-spin" />
      {text}
    </div>
  );
}

export function Notice({ tone, children }: { tone: "error" | "info" | "success"; children: React.ReactNode }) {
  const styles = {
    error: "border-red-800 bg-red-900/20 text-red-300",
    info: "border-slate-700 bg-slate-900/50 text-slate-300",
    success: "border-emerald-800 bg-emerald-900/20 text-emerald-200",
  }[tone];
  return (
    <div role={tone === "error" ? "alert" : "status"} className={`rounded-lg border px-4 py-3 text-sm ${styles}`}>
      {children}
    </div>
  );
}

// The text inputs and selects of the connection form.
export const inputClass =
  "w-full rounded-lg border border-slate-700 bg-slate-900/60 px-3 py-2 text-sm text-white outline-none focus:border-teal-500/60";
