import { Loader2 } from "lucide-react";

// LocalConnectCard is the one-click "Connect local <cloud>" card that
// ConnectLocalAccount and UseLocalDevstack share; each owns what the click does.
export default function LocalConnectCard({
  cloudName,
  onClick,
  busy,
  error,
}: {
  // The cloud's short name ("AWS", "OpenStack").
  cloudName: string;
  onClick: () => void;
  busy: boolean;
  // Why the last click failed, or null.
  error: string | null;
}) {
  return (
    <div className="flex flex-col gap-3 rounded-xl border border-teal-500/40 bg-teal-500/5 p-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="space-y-1">
        <p className="text-sm text-slate-100">Use the {cloudName} running on this machine. One click, nothing to paste.</p>
        {error && (
          <p role="alert" className="text-sm text-red-300">
            {error}
          </p>
        )}
      </div>
      <button
        type="button"
        onClick={onClick}
        disabled={busy}
        className="inline-flex shrink-0 items-center gap-2 rounded-lg bg-teal-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-teal-500 disabled:cursor-not-allowed disabled:opacity-60"
      >
        {busy && <Loader2 className="h-4 w-4 animate-spin" aria-hidden />}
        {busy ? "Connecting…" : `Connect local ${cloudName}`}
      </button>
    </div>
  );
}
