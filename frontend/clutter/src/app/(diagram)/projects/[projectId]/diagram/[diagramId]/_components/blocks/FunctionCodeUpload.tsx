"use client";

import { adapterApiConfigured, adapterErrorMessage, NOT_CONFIGURED_MESSAGE } from "@/lib/adapterApi";
import { useFunctionCode, useRemoveFunctionCode, useUploadFunctionCode } from "@/lib/features/functionCode/hooks";
import { formatBytes, shortHash, uploadReducer, zipFileError } from "@/lib/features/functionCode/upload";
import { useOrganizations } from "@/lib/features/organization/hooks";
import { useProjectOrganization } from "@/lib/features/projects/hooks";
import { useMe } from "@/lib/features/user/hooks";
import { FileArchive, Loader2, RefreshCw, Trash2, Upload } from "lucide-react";
import { useParams } from "next/navigation";
import { useMemo, useReducer, useRef, type ReactNode } from "react";

const buttonClass =
  "inline-flex items-center gap-1.5 rounded-md border border-slate-700 px-2.5 py-1.5 text-xs text-slate-200 hover:bg-slate-800 disabled:opacity-50";

// FunctionCodeUpload is a function block's code: the .zip uploaded through
// B for this block, which every deployment of the diagram uses. With none,
// the function deploys with Clutter's placeholder code.
export default function FunctionCodeUpload({ nodeId, disabled }: { nodeId: string; disabled: boolean }) {
  const params = useParams<{ projectId: string; diagramId: string }>();
  const meQ = useMe();
  const token = meQ.data?.token ?? null;
  const orgsQ = useOrganizations(token);
  const orgIds = useMemo(() => orgsQ.data?.map((o) => o.id) ?? null, [orgsQ.data]);
  // B checks every call against the project's own organization.
  const orgQ = useProjectOrganization(token, params.projectId, orgIds);
  const orgId = orgQ.data ?? null;
  const diagramId = params.diagramId;

  const codeQ = useFunctionCode(token, orgId, diagramId);
  const uploadM = useUploadFunctionCode(token);
  const removeM = useRemoveFunctionCode(token);
  const [state, dispatch] = useReducer(uploadReducer, { phase: "idle" });
  const inputRef = useRef<HTMLInputElement>(null);

  const current = codeQ.data?.find((c) => c.node_id === nodeId) ?? null;
  const busy = state.phase === "uploading" || removeM.isPending;
  const ready = adapterApiConfigured && !!token && !!orgId;

  const choose = (file: File | undefined) => {
    if (!file || !orgId) return;
    const problem = zipFileError(file);
    if (problem) {
      dispatch({ type: "fail", message: problem });
      return;
    }
    dispatch({ type: "start", fileName: file.name });
    removeM.reset();
    uploadM.mutate(
      { organizationId: orgId, diagramId, nodeId, file, onProgress: (loaded, total) => dispatch({ type: "progress", loaded, total }) },
      {
        onSuccess: () => dispatch({ type: "done" }),
        onError: (err) => dispatch({ type: "fail", message: adapterErrorMessage(err) }),
      },
    );
  };

  const remove = () => {
    if (!orgId || !current) return;
    if (!window.confirm(`Remove ${current.file_name}? The function deploys with placeholder code until you upload new code.`)) return;
    dispatch({ type: "dismiss" });
    removeM.mutate({ organizationId: orgId, diagramId, nodeId });
  };

  const openPicker = () => inputRef.current?.click();

  let body: ReactNode;
  if (!adapterApiConfigured) {
    body = <p className="text-xs text-slate-400">{NOT_CONFIGURED_MESSAGE}</p>;
  } else if (orgQ.isError || (orgQ.isSuccess && !orgId)) {
    body = <p className="text-xs text-red-400">Clutter could not find this diagram&apos;s organization. Reload the page.</p>;
  } else if (!ready || codeQ.isPending) {
    body = (
      <p className="flex items-center gap-2 text-xs text-slate-400">
        <Loader2 className="h-3.5 w-3.5 animate-spin" />
        Loading this function&apos;s code…
      </p>
    );
  } else if (codeQ.isError) {
    body = (
      <div className="space-y-1.5">
        <p className="text-xs text-red-400">{adapterErrorMessage(codeQ.error)}</p>
        <button type="button" className={buttonClass} onClick={() => void codeQ.refetch()}>
          <RefreshCw className="h-3.5 w-3.5" />
          Try again
        </button>
      </div>
    );
  } else if (current) {
    body = (
      <div className="space-y-2">
        <div className="flex items-start gap-2 rounded-md border border-slate-700 bg-slate-950/60 p-2">
          <FileArchive className="mt-0.5 h-4 w-4 shrink-0 text-teal-400" />
          <dl className="min-w-0 text-xs">
            <dt className="sr-only">File</dt>
            <dd className="truncate text-slate-200" title={current.file_name}>
              {current.file_name}
            </dd>
            <dt className="sr-only">Size</dt>
            <dd className="text-slate-400">{formatBytes(current.size)}</dd>
            <dt className="sr-only">SHA-256</dt>
            <dd className="font-mono text-slate-400" title={current.sha256}>
              SHA-256 {shortHash(current.sha256)}…
            </dd>
          </dl>
        </div>
        <div className="flex gap-2">
          <button type="button" className={buttonClass} disabled={disabled || busy} onClick={openPicker}>
            <Upload className="h-3.5 w-3.5" />
            Replace
          </button>
          <button
            type="button"
            className={`${buttonClass} border-red-900 text-red-300 hover:bg-red-950`}
            disabled={disabled || busy}
            onClick={remove}>
            {removeM.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Trash2 className="h-3.5 w-3.5" />}
            Remove
          </button>
        </div>
      </div>
    );
  } else {
    body = (
      <div className="space-y-2">
        <p className="text-xs text-slate-400">No code uploaded. The function runs placeholder code until you upload a .zip.</p>
        <button type="button" className={buttonClass} disabled={disabled || busy} onClick={openPicker}>
          <Upload className="h-3.5 w-3.5" />
          Upload .zip
        </button>
      </div>
    );
  }

  return (
    <div className="space-y-1.5">
      <p className="block text-xs font-medium text-slate-300">Code</p>
      <p className="text-xs text-slate-500">A .zip of at most 50 MB, used by every deployment of this diagram. Save the diagram before uploading.</p>
      {body}
      {state.phase === "uploading" && (
        <div className="space-y-1">
          <p className="truncate text-xs text-slate-300">Uploading {state.fileName}…</p>
          <div
            role="progressbar"
            aria-label={`Uploading ${state.fileName}`}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={state.percent}
            className="h-1.5 overflow-hidden rounded bg-slate-800">
            <div className="h-full bg-teal-500 transition-[width]" style={{ width: `${state.percent}%` }} />
          </div>
        </div>
      )}
      {state.phase === "failed" && (
        <div className="flex items-start justify-between gap-2">
          <p role="alert" className="text-xs text-red-400">
            {state.message}
          </p>
          <button type="button" className="text-xs text-slate-400 hover:text-white" onClick={() => dispatch({ type: "dismiss" })}>
            Dismiss
          </button>
        </div>
      )}
      {removeM.isError && (
        <p role="alert" className="text-xs text-red-400">
          {adapterErrorMessage(removeM.error)}
        </p>
      )}
      <input
        ref={inputRef}
        type="file"
        accept=".zip,application/zip"
        className="hidden"
        onChange={(e) => {
          choose(e.target.files?.[0]);
          // Choosing the same file again still uploads it.
          e.target.value = "";
        }}
      />
    </div>
  );
}
