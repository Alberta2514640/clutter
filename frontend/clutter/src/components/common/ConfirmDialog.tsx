"use client";

import { Button } from "@/components/ui/button";
import { AlertTriangle, Loader2, X } from "lucide-react";
import { useEffect, useId, useRef, type ReactNode } from "react";

interface ConfirmDialogProps {
  title: string;
  children: ReactNode;
  confirmLabel: string;
  pendingLabel: string;
  isPending: boolean;
  // A refusal from the last attempt, shown inside the dialog.
  error?: string | null;
  onConfirm: () => void;
  onCancel: () => void;
}

const FOCUSABLE = 'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

// ConfirmDialog asks before a destructive action. It is a modal dialog: focus
// moves into it and stays there, Escape and the backdrop cancel (not while the
// action runs), and focus returns to the opener when it closes.
export default function ConfirmDialog({ title, children, confirmLabel, pendingLabel, isPending, error, onConfirm, onCancel }: ConfirmDialogProps) {
  const titleId = useId();
  const bodyId = useId();
  const panelRef = useRef<HTMLDivElement>(null);
  const cancelRef = useRef<HTMLButtonElement>(null);
  const cancelHandler = useRef(onCancel);
  const pending = useRef(isPending);

  useEffect(() => {
    cancelHandler.current = onCancel;
    pending.current = isPending;
  }, [onCancel, isPending]);

  useEffect(() => {
    const opener = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    cancelRef.current?.focus();

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        if (!pending.current) cancelHandler.current();
        return;
      }
      if (event.key !== "Tab" || !panelRef.current) return;
      const items = Array.from(panelRef.current.querySelectorAll<HTMLElement>(FOCUSABLE));
      if (items.length === 0) return;
      const first = items[0];
      const last = items[items.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    };

    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("keydown", onKeyDown);
      opener?.focus();
    };
  }, []);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={isPending ? undefined : onCancel}>
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={bodyId}
        className="mx-4 w-full max-w-md rounded-xl border border-slate-700 bg-slate-900 p-6 shadow-2xl"
        onClick={(event) => event.stopPropagation()}>
        <div className="mb-6 flex items-start gap-4">
          <div className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg border border-red-800/50 bg-red-900/30">
            <AlertTriangle className="h-5 w-5 text-red-400" aria-hidden="true" />
          </div>
          <div className="min-w-0">
            <h2 id={titleId} className="mb-1 text-base font-semibold text-white">
              {title}
            </h2>
            <div id={bodyId} className="space-y-2 text-sm text-slate-400">
              {children}
            </div>
          </div>
          <button
            type="button"
            onClick={onCancel}
            disabled={isPending}
            aria-label="Close"
            className="ml-auto flex-shrink-0 text-slate-500 transition-colors hover:text-slate-300 disabled:cursor-not-allowed disabled:opacity-50">
            <X className="h-4 w-4" aria-hidden="true" />
          </button>
        </div>

        {error && (
          <p role="alert" className="mb-4 rounded-lg border border-red-800 bg-red-900/20 p-3 text-sm text-red-300">
            {error}
          </p>
        )}

        <div className="flex justify-end gap-3">
          <Button
            ref={cancelRef}
            type="button"
            onClick={onCancel}
            disabled={isPending}
            className="h-9 rounded-lg border border-slate-700 bg-slate-800 px-4 text-sm font-medium text-slate-300 hover:bg-slate-700">
            Cancel
          </Button>
          <Button type="button" onClick={onConfirm} disabled={isPending} className="h-9 rounded-lg bg-red-600 px-4 text-sm font-medium text-white hover:bg-red-500">
            {isPending ? (
              <>
                <Loader2 className="mr-2 h-3.5 w-3.5 animate-spin" aria-hidden="true" />
                {pendingLabel}
              </>
            ) : (
              confirmLabel
            )}
          </Button>
        </div>
      </div>
    </div>
  );
}
