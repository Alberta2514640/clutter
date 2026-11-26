"use client";

import ErrorState from "@/components/common/ErrorState";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { toast } from "@/components/ui/sonner";
import { adapterApiConfigured, adapterErrorMessage, NOT_CONFIGURED_MESSAGE } from "@/lib/adapterApi";
import { useFunctionCode, useRemoveFunctionCode, useUploadFunctionCode } from "@/lib/features/functionCode/hooks";
import { MAX_CODE_SIZE_TEXT } from "@/lib/features/functionCode/types";
import { formatBytes, shortHash, uploadReducer, zipFileError } from "@/lib/features/functionCode/upload";
import { FileArchive, Loader2, Trash2, Upload } from "lucide-react";
import { useReducer, useRef, useState, type ReactNode } from "react";

type Props = {
  nodeId: string;
  token: string | null;
  // The project's own organization, which B checks every call against, and
  // why it cannot be found (null while it is found or still being looked up).
  organizationId: string | null;
  organizationError: string | null;
  diagramId: string;
};

// FunctionCodeUpload is a function block's code: the .zip uploaded through
// B for this block, which every deployment of the diagram uses. With none,
// the function deploys with Clutter's sample code.
export default function FunctionCodeUpload({ nodeId, token, organizationId: orgId, organizationError, diagramId }: Props) {
  const codeQ = useFunctionCode(token, orgId, diagramId);
  const uploadM = useUploadFunctionCode(token);
  const removeM = useRemoveFunctionCode(token);
  const [state, dispatch] = useReducer(uploadReducer, { phase: "idle" });
  const [confirmRemove, setConfirmRemove] = useState(false);
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
        onSuccess: () => {
          dispatch({ type: "done" });
          toast.success("Code uploaded");
        },
        onError: (err) => dispatch({ type: "fail", message: adapterErrorMessage(err) }),
      },
    );
  };

  const remove = () => {
    if (!orgId || !current) return;
    dispatch({ type: "dismiss" });
    removeM.mutate({ organizationId: orgId, diagramId, nodeId }, { onSuccess: () => toast.success("Code removed") });
  };

  const openPicker = () => inputRef.current?.click();

  let body: ReactNode;
  if (!adapterApiConfigured) {
    body = <p className="text-muted-foreground text-xs">{NOT_CONFIGURED_MESSAGE}</p>;
  } else if (organizationError) {
    body = <p className="text-danger text-xs">{organizationError}</p>;
  } else if (!ready || codeQ.isPending) {
    body = (
      <p role="status" className="text-muted-foreground flex items-center gap-2 text-xs">
        <Loader2 className="size-3.5 animate-spin" aria-hidden />
        Loading this function&apos;s code…
      </p>
    );
  } else if (codeQ.isError && !codeQ.data) {
    body = <ErrorState variant="inline" description={`Could not read this function's code. ${adapterErrorMessage(codeQ.error)}`} onRetry={() => void codeQ.refetch()} />;
  } else if (current) {
    body = (
      <div className="space-y-2">
        <div className="bg-muted/50 flex items-start gap-2 rounded-md border p-2">
          <FileArchive className="text-link mt-0.5 size-4 shrink-0" aria-hidden />
          <dl className="min-w-0 text-xs">
            <dt className="sr-only">File</dt>
            <dd className="truncate font-medium" title={current.file_name}>
              {current.file_name}
            </dd>
            <dt className="sr-only">Size</dt>
            <dd className="text-muted-foreground">{formatBytes(current.size)}</dd>
            <dt className="sr-only">SHA-256</dt>
            <dd className="text-muted-foreground font-mono" title={current.sha256}>
              <span aria-hidden="true">SHA-256 </span>
              {shortHash(current.sha256)}…
            </dd>
          </dl>
        </div>
        <div className="flex gap-2">
          <Button type="button" variant="outline" size="xs" disabled={busy} onClick={openPicker}>
            <Upload aria-hidden />
            Replace
          </Button>
          <Button type="button" variant="outline" size="xs" className="text-danger" disabled={busy} onClick={() => setConfirmRemove(true)}>
            {removeM.isPending ? <Loader2 className="animate-spin" aria-hidden /> : <Trash2 aria-hidden />}
            Remove
          </Button>
        </div>
      </div>
    );
  } else {
    body = (
      <div className="space-y-2">
        <p className="text-muted-foreground text-xs">No code uploaded. The function runs sample code until you upload a .zip.</p>
        <Button type="button" variant="outline" size="xs" disabled={busy} onClick={openPicker}>
          <Upload aria-hidden />
          Upload .zip
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-1.5">
      <p className="block text-xs font-medium">Code</p>
      <p className="text-muted-foreground text-xs">A .zip of at most {MAX_CODE_SIZE_TEXT}, used by every deployment of this diagram. Save the diagram before uploading.</p>
      {body}
      {state.phase === "uploading" && (
        <div className="space-y-1">
          <p className="text-muted-foreground truncate text-xs">Uploading {state.fileName}…</p>
          <div
            role="progressbar"
            aria-label={`Uploading ${state.fileName}`}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={state.percent}
            className="bg-muted h-1.5 overflow-hidden rounded">
            <div className="bg-primary h-full origin-left transition-transform" style={{ transform: `scaleX(${state.percent / 100})` }} />
          </div>
        </div>
      )}
      {state.phase === "failed" && (
        <div className="flex items-start justify-between gap-2">
          <p role="alert" className="text-danger text-xs">
            {state.message}
          </p>
          <Button type="button" variant="ghost" size="xs" onClick={() => dispatch({ type: "dismiss" })}>
            Dismiss
          </Button>
        </div>
      )}
      {removeM.isError && (
        <p role="alert" className="text-danger text-xs">
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
      <AlertDialog open={confirmRemove} onOpenChange={setConfirmRemove}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove {current?.file_name ?? "this code"}?</AlertDialogTitle>
            <AlertDialogDescription>The function deploys with sample code until you upload new code.</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Keep it</AlertDialogCancel>
            <AlertDialogAction onClick={remove}>Remove code</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
