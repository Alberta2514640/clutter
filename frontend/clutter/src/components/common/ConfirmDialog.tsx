"use client";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { AlertTriangle, Loader2 } from "lucide-react";
import type { ReactNode } from "react";

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

// ConfirmDialog asks before a destructive action, on the one AlertDialog. The
// caller mounts it while it is open. Cancel has focus, Escape cancels (not
// while the action runs), and focus returns to the opener when it closes. The
// confirm button runs onConfirm and leaves closing to the caller, so the
// dialog stays open with the refusal when the action fails.
export default function ConfirmDialog({ title, children, confirmLabel, pendingLabel, isPending, error, onConfirm, onCancel }: ConfirmDialogProps) {
  return (
    <AlertDialog
      open
      onOpenChange={(open) => {
        if (!open && !isPending) onCancel();
      }}>
      <AlertDialogContent>
        <div className="flex items-start gap-4">
          <span className="bg-danger/10 text-danger flex size-10 shrink-0 items-center justify-center rounded-lg">
            <AlertTriangle className="size-5" aria-hidden="true" />
          </span>
          <div className="min-w-0 space-y-1.5">
            <AlertDialogTitle>{title}</AlertDialogTitle>
            <AlertDialogDescription asChild>
              <div className="space-y-2">{children}</div>
            </AlertDialogDescription>
          </div>
        </div>

        {error && (
          <p role="alert" className="border-danger/30 bg-danger/10 text-danger rounded-lg border p-3 text-sm">
            {error}
          </p>
        )}

        <AlertDialogFooter>
          <AlertDialogCancel disabled={isPending}>Cancel</AlertDialogCancel>
          <AlertDialogAction
            disabled={isPending}
            onClick={(event) => {
              event.preventDefault();
              onConfirm();
            }}>
            {isPending ? (
              <>
                <Loader2 className="animate-spin" aria-hidden="true" />
                {pendingLabel}
              </>
            ) : (
              confirmLabel
            )}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
