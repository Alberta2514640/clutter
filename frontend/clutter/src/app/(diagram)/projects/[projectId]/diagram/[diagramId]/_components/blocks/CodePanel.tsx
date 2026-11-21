"use client";

import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { AdapterApiError, adapterErrorMessage } from "@/lib/adapterApi";
import { usePreview } from "@/lib/features/deployments/hooks";
import { previewGroups } from "@/lib/features/deployments/preview";
import { usePreferencesStore } from "@/lib/features/preferences/uiStore";
import { joinAnd } from "@/lib/text";
import { cn } from "@/lib/utils";
import { Check, Code2, Copy, Download, Maximize2, Minimize2, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";

type Props = {
  token: string | null;
  organizationId: string | null;
  // The deployment whose Terraform to show, or null when the diagram has none.
  deployment: { id: string; title: string } | null;
  // Set while the canvas has unsaved changes: the preview shows the saved diagram.
  unsaved: boolean;
  onClose: () => void;
  onAddDeployment: () => void;
};

// The code panel: the Terraform one deployment would run, rendered by its
// cloud's connector without credentials, in a drawer over the canvas. A
// split deployment's files are grouped by the connection that builds them.
// Opening it on a deployment ticks "Preview the code" on the checklist.
export default function CodePanel({ token, organizationId, deployment, unsaved, onClose, onAddDeployment }: Props) {
  const [tall, setTall] = useState(false);

  // Escape closes the panel, unless a dialog or menu above it takes the key.
  const closeRef = useRef(onClose);
  useEffect(() => {
    closeRef.current = onClose;
  });
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== "Escape" || e.defaultPrevented) return;
      if (document.querySelector('[role="dialog"], [role="alertdialog"], [role="menu"]')) return;
      closeRef.current();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);
  return (
    <section
      aria-label="Terraform preview"
      data-testid="code-panel"
      className={cn(
        "bg-background absolute inset-x-0 bottom-0 z-30 flex flex-col border-t shadow-[0_-8px_24px_-12px_rgb(0_0_0/0.25)] motion-slow animate-in slide-in-from-bottom-4 fade-in-0",
        tall ? "top-0" : "h-[45%] min-h-64",
      )}>
      <div className="flex h-11 shrink-0 items-center gap-2 border-b px-3">
        <Code2 className="text-link size-4" aria-hidden />
        <h2 className="min-w-0 truncate text-sm font-semibold">{deployment ? `Terraform for ${deployment.title}` : "Terraform preview"}</h2>
        {unsaved && deployment && <span className="text-warning text-xs">Shows the saved diagram. Save to see your changes.</span>}
        <div className="ml-auto flex items-center gap-1">
          <Button type="button" variant="ghost" size="icon-sm" onClick={() => setTall((t) => !t)} aria-label={tall ? "Make the code panel smaller" : "Make the code panel taller"}>
            {tall ? <Minimize2 aria-hidden /> : <Maximize2 aria-hidden />}
          </Button>
          <Button type="button" variant="ghost" size="icon-sm" onClick={onClose} aria-label="Close the code panel" title="Close (Esc)">
            <X aria-hidden />
          </Button>
        </div>
      </div>
      {deployment ? (
        <PreviewFiles key={deployment.id} token={token} organizationId={organizationId} deploymentId={deployment.id} />
      ) : (
        <div className="flex flex-1 flex-col items-center justify-center gap-3 p-6 text-center">
          <p className="text-muted-foreground max-w-sm text-sm">Clutter writes Terraform for each deployment. Add a deployment to see the code for that cloud.</p>
          <Button type="button" size="sm" onClick={onAddDeployment}>
            Add a deployment
          </Button>
        </div>
      )}
    </section>
  );
}

function PreviewFiles({ token, organizationId, deploymentId }: { token: string | null; organizationId: string | null; deploymentId: string }) {
  const previewQ = usePreview(token, organizationId, deploymentId);
  const markPreviewOpened = usePreferencesStore((s) => s.markPreviewOpened);
  const files = previewQ.data ?? {};
  const groups = previewGroups(Object.keys(files));
  const [pickedGroup, setPickedGroup] = useState<string | null>(null);
  const group = groups.find((g) => g.connection === pickedGroup) ?? groups[0] ?? null;
  const [pickedFile, setPickedFile] = useState<string | null>(null);
  // main.tf first: it holds the blocks, while the files before it in name
  // order (defaults.tf) hold only settings.
  const current = group?.files.find((f) => f.path === pickedFile) ?? group?.files.find((f) => f.name === "main.tf") ?? group?.files[0] ?? null;
  const [copied, setCopied] = useState<"copied" | "failed" | null>(null);
  const named = groups.filter((g) => g.connection).map((g) => g.connection as string);
  const text = current ? (files[current.path] ?? "") : "";

  // Opening the code panel at all does the checklist's step, as the card's
  // Preview code button does, even on a cloud that offers no preview.
  useEffect(() => {
    if (organizationId) markPreviewOpened(organizationId);
  }, [organizationId, markPreviewOpened]);

  useEffect(() => {
    if (!copied) return;
    const t = window.setTimeout(() => setCopied(null), 2000);
    return () => window.clearTimeout(t);
  }, [copied]);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied("copied");
    } catch {
      setCopied("failed");
    }
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

  if (previewQ.isLoading) {
    return (
      <div role="status" className="flex-1 space-y-2 p-4">
        <span className="sr-only">Writing the Terraform…</span>
        <Skeleton className="h-4 w-1/3" />
        <Skeleton className="h-4 w-2/3" />
        <Skeleton className="h-4 w-1/2" />
      </div>
    );
  }
  if (previewQ.isError) {
    return (
      <div className="flex-1 overflow-auto p-4">
        <p role="alert" className={cn("text-sm", previewQ.error instanceof AdapterApiError && previewQ.error.status === 501 ? "text-muted-foreground" : "text-danger")}>
          {adapterErrorMessage(previewQ.error)}
        </p>
        {previewQ.error instanceof AdapterApiError && previewQ.error.problems.length > 0 && (
          <ul className="text-danger mt-2 list-disc pl-5 text-sm">
            {previewQ.error.problems.map((p, i) => (
              <li key={i}>{p.message}</li>
            ))}
          </ul>
        )}
      </div>
    );
  }
  if (!group || !current) return <p className="text-muted-foreground flex-1 p-4 text-sm">This deployment renders no files.</p>;

  const tab = (on: boolean) =>
    cn(
      "h-8 shrink-0 rounded-md px-2.5 text-xs whitespace-nowrap outline-none focus-visible:ring-ring/50 focus-visible:ring-[3px] transition-colors",
      on ? "bg-secondary text-foreground font-medium" : "text-muted-foreground hover:text-foreground",
    );

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      {named.length > 1 && (
        <div className="flex items-center gap-1 overflow-x-auto overflow-y-hidden border-b px-3 py-1.5" role="tablist" aria-label={`Connection: ${joinAnd(named)}`}>
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
              className={tab(g === group)}>
              {g.connection} <span className="text-muted-foreground">({g.files.length})</span>
            </button>
          ))}
        </div>
      )}
      <div className="flex items-center gap-2 border-b px-3 py-1.5">
        <div className="flex min-w-0 flex-1 gap-1 overflow-x-auto overflow-y-hidden" role="tablist" aria-label="File">
          {group.files.map((f) => (
            <button key={f.path} type="button" role="tab" aria-selected={f === current} onClick={() => setPickedFile(f.path)} className={cn(tab(f === current), "font-mono")}>
              {f.name}
            </button>
          ))}
        </div>
        <Button type="button" variant="outline" size="xs" onClick={() => void copy()}>
          {copied === "copied" ? <Check aria-hidden /> : <Copy aria-hidden />}
          <span aria-live="polite">{copied === "copied" ? "Copied" : copied === "failed" ? "Could not copy" : "Copy"}</span>
        </Button>
        <Button type="button" variant="outline" size="xs" onClick={download}>
          <Download aria-hidden />
          Download
        </Button>
      </div>
      <pre
        data-testid="preview-file"
        tabIndex={0}
        aria-label={`Contents of ${current.path}`}
        className="bg-muted/40 focus-visible:ring-ring/50 min-h-0 flex-1 overflow-auto p-4 font-mono text-xs leading-relaxed whitespace-pre outline-none focus-visible:ring-[3px] focus-visible:ring-inset">
        {text}
      </pre>
    </div>
  );
}
