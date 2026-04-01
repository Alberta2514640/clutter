"use client";

import { AlertTriangle } from "lucide-react";

type AwsAccountRequiredModalProps = {
  open: boolean;
  onGoToSettings: () => void;
  onBack: () => void;
};

export default function AwsAccountRequiredModal({
  open,
  onGoToSettings,
  onBack,
}: AwsAccountRequiredModalProps) {
  if (!open) return null;

  return (
    <div className="absolute inset-0 z-[100] flex items-center justify-center bg-black/70 backdrop-blur-sm">
      <div className="mx-4 w-full max-w-md rounded-2xl border border-amber-500/30 bg-neutral-900 p-6 shadow-2xl">
        <div className="mb-4 flex items-start gap-3">
          <div className="rounded-full bg-amber-500/15 p-2">
            <AlertTriangle className="h-5 w-5 text-amber-400" />
          </div>

          <div>
            <h2 className="text-lg font-semibold text-white">
              AWS account required
            </h2>
            <p className="mt-1 text-sm text-slate-300">
              You need to link an AWS account before using this diagram editor.
            </p>
          </div>
        </div>

        <div className="rounded-lg border border-slate-800 bg-slate-950/60 p-3 text-sm text-slate-300">
          Go to{" "}
          <span className="font-medium text-white">
            Settings → Organization → AWS Account
          </span>{" "}
          to connect your AWS account.
        </div>

        <div className="mt-6 flex items-center justify-end gap-3">
          <button
            type="button"
            onClick={onBack}
            className="rounded-lg border border-slate-700 px-4 py-2 text-sm font-medium text-slate-200 transition hover:bg-slate-800"
          >
            Back
          </button>

          <button
            type="button"
            onClick={onGoToSettings}
            className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-blue-500"
          >
            Link AWS account
          </button>
        </div>
      </div>
    </div>
  );
}