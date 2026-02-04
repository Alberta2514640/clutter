"use client";

import { Button } from "@/components/ui/button";
import { useRouter } from "next/navigation";

export default function TopNav({ onSave, onBack, dirty, isSaving, }: { onSave: () => void; onBack?: () => void; dirty: boolean; isSaving?: boolean; }) {
  const router = useRouter();

  const canSave = dirty && !isSaving;

  return (
    <div className="flex justify-end gap-4 px-5 py-5">
      <Button
        onClick={onBack ?? (() => router.back())}
        className="rounded-lg border border-white/10 bg-white/5 px-5 py-2 text-sm font-semibold text-white shadow-lg backdrop-blur-sm transition-colors hover:bg-white/10"
      >
        ← Back
      </Button>

      <Button
        onClick={onSave}
        disabled={!canSave}
        className={[
          "px-6 py-2 rounded-lg font-semibold shadow-lg text-white",
          canSave
            ? "bg-gradient-to-br from-teal-600 to-blue-600 hover:opacity-90"
            : "bg-white/10 text-white/60 cursor-not-allowed",
        ].join(" ")}
      >
        {isSaving ? "Saving…" : dirty ? "Save" : "Saved"}
      </Button>
    </div>
  );
}