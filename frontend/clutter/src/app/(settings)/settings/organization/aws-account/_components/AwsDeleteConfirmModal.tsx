"use client";

import { AlertTriangle, X } from "lucide-react";

interface AwsDeleteConfirmModalProps {
  accountName: string;
  isDeleting: boolean;
  onCancel: () => void;
  onConfirm: () => Promise<void>;
}

export default function AwsDeleteConfirmModal({
  accountName,
  isDeleting,
  onCancel,
  onConfirm,
}: AwsDeleteConfirmModalProps) {
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
      onClick={isDeleting ? undefined : onCancel}
    >
      <div
        className="mx-4 w-full max-w-2xl rounded-[28px] border border-slate-700 bg-slate-900 p-8 shadow-2xl"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="mb-8 flex items-start gap-5">
          <div className="flex h-14 w-14 flex-shrink-0 items-center justify-center rounded-2xl border border-amber-800/60 bg-amber-900/20">
            <AlertTriangle className="h-7 w-7 text-amber-400" />
          </div>
          <div className="min-w-0 flex-1">
            <h3 className="mb-2 text-3xl font-semibold tracking-tight text-white">
              Delete AWS account link?
            </h3>
            <p className="text-xl leading-relaxed text-slate-400">
              You are about to remove{" "}
              <span className="font-semibold text-white">
                &quot;{accountName}&quot;
              </span>
              . If you continue, this AWS account connection will be deleted and you will need to set it up again.
            </p>
          </div>
          <button
            type="button"
            onClick={onCancel}
            disabled={isDeleting}
            className="flex-shrink-0 text-slate-500 transition-colors hover:text-slate-300 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <X className="h-6 w-6" />
          </button>
        </div>

        <div className="flex justify-end gap-4">
          <button
            type="button"
            onClick={onCancel}
            disabled={isDeleting}
            className="h-14 rounded-2xl border border-slate-700 bg-slate-800 px-8 text-2xl font-semibold text-slate-300 transition-colors hover:bg-slate-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={() => void onConfirm()}
            disabled={isDeleting}
            className="h-14 rounded-2xl bg-amber-600 px-8 text-2xl font-semibold text-white transition-colors hover:bg-amber-500 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {isDeleting ? "Deleting..." : "Delete"}
          </button>
        </div>
      </div>
    </div>
  );
}
