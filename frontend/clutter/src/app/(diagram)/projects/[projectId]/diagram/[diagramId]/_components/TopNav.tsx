"use client";

import { Button } from "@/components/ui/button";
import { useCallback, useEffect, useRef, useState } from "react";

type TopNavProps = {
  diagramName: string;
  onNameChange: (nextName: string) => void;
  onSave: () => void;
  onBack?: () => void;
  dirty: boolean;
  isSaving?: boolean;
  nameDisabled?: boolean;
};

export default function TopNav({diagramName, onNameChange, onSave, onBack, dirty, isSaving, nameDisabled}: TopNavProps) {
  const canSave = dirty && !isSaving;

  const [isEditing, setIsEditing] = useState(false);
  const [draft, setDraft] = useState(""); // only meaningful while editing
  const inputRef = useRef<HTMLInputElement | null>(null);

  // focus when we enter edit mode
  useEffect(() => {
    if (!isEditing) return;
    queueMicrotask(() => {
      inputRef.current?.focus();
      inputRef.current?.select();
    });
  }, [isEditing]);

  const startEdit = useCallback(() => {
    if (nameDisabled) return;
    setDraft(diagramName);   // initialize from store at the moment editing begins
    setIsEditing(true);
  }, [diagramName, nameDisabled]);

  const commit = useCallback(() => {
    const next = draft.trim();

    // empty => revert
    if (!next) {
      setIsEditing(false);
      return;
    }

    if (next !== diagramName) {
      onNameChange(next);
    }
    setIsEditing(false);
  }, [draft, diagramName, onNameChange]);

  const cancel = useCallback(() => {
    setIsEditing(false);
  }, []);

  return (
    <div className="flex items-center justify-between gap-4 px-5 py-5">
      {/* Left: Diagram name (click to edit) */}
      <div className="min-w-0">
        {!isEditing ? (
          <button
            type="button"
            disabled={!!nameDisabled}
            onClick={startEdit}
            className={[
              // Palette-like styling
              "h-10 max-w-[420px] truncate rounded-lg border border-slate-800 bg-slate-950/70 px-4",
              "text-sm font-semibold text-slate-100 shadow-lg backdrop-blur-sm",
              "transition-colors hover:bg-slate-900/70",
              "disabled:cursor-not-allowed disabled:opacity-60",
            ].join(" ")}
            title="Click to rename"
          >
            {diagramName || "Untitled diagram"}
          </button>
        ) : (
          <input
            ref={inputRef}
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onBlur={commit}
            onKeyDown={(e) => {
              if (e.key === "Enter") commit();
              if (e.key === "Escape") cancel();
            }}
            className={[
              "h-10 w-[min(420px,60vw)] rounded-lg border border-slate-800 bg-slate-950/70 px-4",
              "text-sm font-semibold text-slate-100 shadow-lg backdrop-blur-sm outline-none",
              "focus:border-slate-700",
            ].join(" ")}
            placeholder="Diagram name"
          />
        )}
      </div>

      {/* Right: Back + Save */}
      <div className="flex items-center gap-4">
        <Button
          onClick={onBack}
          className="h-10 rounded-lg border border-white/10 bg-white/5 px-5 text-sm font-semibold text-white shadow-lg backdrop-blur-sm transition-colors hover:bg-white/10"
        >
          ← Back
        </Button>

        <Button
          onClick={onSave}
          disabled={!canSave}
          className={[
            "h-10 px-6 rounded-lg font-semibold shadow-lg text-white",
            canSave
              ? "bg-gradient-to-br from-teal-600 to-blue-600 hover:opacity-90"
              : "bg-white/10 text-white/60 cursor-not-allowed",
          ].join(" ")}
        >
          {isSaving ? "Saving…" : dirty ? "Save" : "Saved"}
        </Button>
      </div>
    </div>
  );
}