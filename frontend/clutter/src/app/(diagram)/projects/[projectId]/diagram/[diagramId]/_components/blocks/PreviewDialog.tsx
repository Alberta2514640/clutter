"use client";

import { AdapterApiError, adapterErrorMessage } from "@/lib/adapterApi";
import { usePreview } from "@/lib/features/deployments/hooks";
import { joinAnd, previewGroups } from "@/lib/features/deployments/preview";
import { Check, Copy, Download, Loader2, X } from "lucide-react";
import { useId, useRef, useState } from "react";
import { useDialogFocus } from "@/components/common/useDialogFocus";

// The Terraform one deployment would run, rendered by its cloud's connector
// without credentials. A split deployment's files are grouped by the
// connection that builds them. A modal: focus moves in and stays, Escape
// closes it.
export default function PreviewDialog({
  token,
  organizationId,
  deploymentId,
  title,
  onClose,
}: {
  token: string | null;
  organizationId: string | null;
  deploymentId: string;
  // What a one-cloud deployment is called, such as "Prod (Amazon Web Services)".
  title: string;
  onClose: () => void;
}) {
  const previewQ = usePreview(token, organizationId, deploymentId, true);
  const files = previewQ.data ?? {};
  const groups = previewGroups(Object.keys(files));
  const [pickedGroup, setPickedGroup] = useState<string | null>(null);
  const group = groups.find((g) => g.connection === pickedGroup) ?? groups[0] ?? null;
  const [pickedFile, setPickedFile] = useState<string | null>(null);
  const current = group?.files.find((f) => f.path === pickedFile) ?? group?.files[0] ?? null;
  const [copied, setCopied] = useState<"copied" | "failed" | null>(null);
  const dialogRef = useRef<HTMLDivElement>(null);
  const titleId = useId();
  useDialogFocus(dialogRef, onClose);

  const named = groups.filter((g) => g.connection).map((g) => g.connection as string);
  const heading = named.length > 1 ? `Terraform for ${joinAnd(named)}` : `Terraform for ${title}`;
  const text = current ? (files[current.path] ?? "") : "";

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied("copied");
    } catch {
      setCopied("failed");
    }
    setTimeout(() => setCopied(null), 2000);
  };

  const download = () => {
    if (!current) return;
    const url = URL.createObjectURL(new Blob([text], { type: "text/plain" }));
    const a = document.createElement("a");
    a.href = url;
    a.download = current.name;
    a.click();
    URL.revokeObjectURL(url);
  };

  const tabClass = (on: boolean) =>
    `shrink-0 whitespace-nowrap rounded-t-md px-3 py-1.5 text-xs focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-teal-300 ${
      on ? "bg-slate-800 text-white" : "text-slate-400 hover:text-white"
    }`;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-6"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}>
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
        className="flex h-[80vh] w-full max-w-4xl flex-col rounded-xl border border-slate-700 bg-slate-950 shadow-2xl outline-none">
        <div className="flex items-center justify-between gap-3 border-b border-slate-800 px-5 py-3">
          <h2 id={titleId} className="min-w-0 truncate font-semibold text-white" title={heading}>
            {heading}
          </h2>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close the Terraform preview"
            className="shrink-0 rounded p-1 text-slate-400 hover:bg-slate-800 hover:text-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-teal-300">
            <X className="h-5 w-5" aria-hidden />
          </button>
        </div>
        {previewQ.isLoading ? (
          <div role="status" className="flex flex-1 items-center justify-center gap-2 text-sm text-slate-400">
            <Loader2 className="h-4 w-4 animate-spin" />
            Rendering…
          </div>
        ) : previewQ.isError ? (
          <div className="flex-1 p-5">
            <p role="alert" className="text-sm text-red-300">
              {adapterErrorMessage(previewQ.error)}
            </p>
            {previewQ.error instanceof AdapterApiError && previewQ.error.problems.length > 0 && (
              <ul className="mt-2 list-disc pl-5 text-sm text-red-300">
                {previewQ.error.problems.map((p, i) => (
                  <li key={i}>{p.message}</li>
                ))}
              </ul>
            )}
          </div>
        ) : !group || !current ? (
          <p className="flex-1 p-5 text-sm text-slate-400">This deployment renders no files.</p>
        ) : (
          <>
            {named.length > 1 && (
              <div className="flex items-center gap-2 overflow-x-auto border-b border-slate-800 px-4 py-2" role="tablist" aria-label="Connection">
                <span className="shrink-0 text-xs text-slate-500">Connection</span>
                {groups.map((g) => (
                  <button
                    key={g.connection ?? ""}
                    type="button"
                    role="tab"
                    aria-selected={g === group}
                    onClick={() => {
                      setPickedGroup(g.connection);
                      setPickedFile(null);
                    }}
                    className={`shrink-0 whitespace-nowrap rounded-md border px-2.5 py-1 text-xs focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-teal-300 ${
                      g === group ? "border-teal-500 bg-teal-500/10 text-white" : "border-slate-700 text-slate-300 hover:text-white"
                    }`}>
                    {g.connection} <span className="text-slate-500">({g.files.length})</span>
                  </button>
                ))}
              </div>
            )}
            <div className="flex items-end gap-2 border-b border-slate-800 px-3 pt-2">
              <div className="flex min-w-0 flex-1 gap-1 overflow-x-auto" role="tablist" aria-label="File">
                {group.files.map((f) => (
                  <button key={f.path} type="button" role="tab" aria-selected={f === current} onClick={() => setPickedFile(f.path)} className={`${tabClass(f === current)} font-mono`}>
                    {f.name}
                  </button>
                ))}
              </div>
              <div className="flex shrink-0 items-center gap-1 pb-1">
                <button
                  type="button"
                  onClick={() => void copy()}
                  className="inline-flex items-center gap-1 rounded-md border border-slate-700 px-2 py-1 text-xs text-slate-200 hover:bg-slate-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-teal-300">
                  {copied === "copied" ? <Check className="h-3.5 w-3.5" aria-hidden /> : <Copy className="h-3.5 w-3.5" aria-hidden />}
                  {copied === "copied" ? "Copied" : copied === "failed" ? "Could not copy" : "Copy"}
                </button>
                <button
                  type="button"
                  onClick={download}
                  className="inline-flex items-center gap-1 rounded-md border border-slate-700 px-2 py-1 text-xs text-slate-200 hover:bg-slate-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-teal-300">
                  <Download className="h-3.5 w-3.5" aria-hidden />
                  Download
                </button>
              </div>
            </div>
            <pre
              data-testid="preview-file"
              tabIndex={0}
              aria-label={`Contents of ${current.path}`}
              className="flex-1 overflow-auto whitespace-pre p-4 font-mono text-xs text-slate-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-teal-300">
              {text}
            </pre>
          </>
        )}
      </div>
    </div>
  );
}
