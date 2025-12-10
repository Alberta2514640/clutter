"use client";

import { useRouter } from "next/navigation";

export default function TopNav({ onSave, onBack }: { onSave: () => void; onBack?: () => void; }) {
  const router = useRouter();

  return (
    <div className="mb-4 flex items-center justify-between">
      <button
        type="button"
        onClick={onBack ?? (() => router.back())}
        className="rounded-lg border border-white/10 bg-white/5 px-4 py-2 text-sm font-semibold text-white shadow-lg backdrop-blur-sm transition-colors hover:bg-white/10"
      >
        ← Back
      </button>

      <button
        type="button"
        onClick={onSave}
        className="rounded-lg border border-blue-400/20 bg-blue-500/20 px-5 py-2 text-sm font-semibold text-white shadow-lg backdrop-blur-sm transition-colors hover:bg-blue-500/30"
      >
        Save
      </button>
    </div>
  );
}