"use client";

import * as React from "react";
import { Undo2, Redo2, ZoomIn, ZoomOut, Maximize2, LayoutPanelTop, Download, Play, CheckCircle2, Lock, Unlock, Cloud, Loader2, ShieldAlert } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

export type ToolbarProps = {
  // Editing & lock
  canEdit: boolean;
  editing: boolean;
  lockOwner?: string | null;
  onAcquireLock?: () => void;
  onReleaseLock?: () => void;

  // Undo/redo
  canUndo?: boolean;
  canRedo?: boolean;
  onUndo?: () => void;
  onRedo?: () => void;

  // Zoom & layout
  onZoomIn?: () => void;
  onZoomOut?: () => void;
  onZoomReset?: () => void;
  onAutoLayout?: () => void;

  // Export TF
  exporting?: boolean;
  onExportTerraform?: () => void;

  // Runs (workspace-bound)
  workspaceBound?: boolean;
  runInProgress?: boolean;
  onPlan?: () => void;
  onApply?: () => void;

  // Save status
  saving?: boolean;
  dirty?: boolean;

  // Optional className
  className?: string;
};

/**
 * Diagram Toolbar
 * - Left: Undo/Redo · Zoom · Auto-layout
 * - Center: Save status
 * - Right: Lock controls · Export Terraform · Plan/Apply (if workspace bound)
 */
export default function Toolbar({
  canEdit,
  editing,
  lockOwner,
  onAcquireLock,
  onReleaseLock,
  canUndo = false,
  canRedo = false,
  onUndo,
  onRedo,
  onZoomIn,
  onZoomOut,
  onZoomReset,
  onAutoLayout,
  exporting = false,
  onExportTerraform,
  workspaceBound = false,
  runInProgress = false,
  onPlan,
  onApply,
  saving = false,
  dirty = false,
  className,
}: ToolbarProps) {
  const viewOnly = !editing;

  return (
    <TooltipProvider delayDuration={150}>
      <div className={cn("flex w-full items-center justify-between gap-2 border-b bg-background/80 p-2 backdrop-blur supports-[backdrop-filter]:bg-background/60", className)} role="toolbar" aria-label="Diagram toolbar">
        {/* LEFT GROUP — Edit & View Controls */}
        <div className="flex items-center gap-1">
          <ToolTipButton ariaLabel="Undo" label="Undo" kbd={["⌘", "Z"]} disabled={!canEdit || viewOnly || !canUndo} onClick={onUndo}>
            <Undo2 className="h-4 w-4" />
          </ToolTipButton>
          <ToolTipButton ariaLabel="Redo" label="Redo" kbd={["⌘", "⇧", "Z"]} disabled={!canEdit || viewOnly || !canRedo} onClick={onRedo}>
            <Redo2 className="h-4 w-4" />
          </ToolTipButton>

          <Separator orientation="vertical" className="mx-2 h-6" />

          <ToolTipButton ariaLabel="Zoom out" label="Zoom out" kbd={["-"]} onClick={onZoomOut}>
            <ZoomOut className="h-4 w-4" />
          </ToolTipButton>
          <ToolTipButton ariaLabel="Reset zoom" label="Reset zoom" kbd={["0"]} onClick={onZoomReset}>
            <Maximize2 className="h-4 w-4" />
          </ToolTipButton>
          <ToolTipButton ariaLabel="Zoom in" label="Zoom in" kbd={["+"]} onClick={onZoomIn}>
            <ZoomIn className="h-4 w-4" />
          </ToolTipButton>

          <Separator orientation="vertical" className="mx-2 h-6" />

          <ToolTipButton ariaLabel="Auto-layout" label="Auto-layout" description="Reflows nodes using ELK (non-destructive)." disabled={!canEdit || viewOnly} onClick={onAutoLayout}>
            <LayoutPanelTop className="h-4 w-4" />
          </ToolTipButton>
        </div>

        {/* CENTER — Save state */}
        <div className="flex items-center gap-2">
          {saving ? (
            <Badge variant="secondary" className="flex items-center gap-1 text-muted-foreground" aria-live="polite">
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
              Saving…
            </Badge>
          ) : dirty ? (
            <Badge variant="secondary" className="text-amber-700 dark:text-amber-300" aria-live="polite">
              Unsaved changes
            </Badge>
          ) : (
            <Badge variant="secondary" className="flex items-center gap-1 text-emerald-700 dark:text-emerald-300" aria-live="polite">
              <CheckCircle2 className="h-3.5 w-3.5" />
              Saved
            </Badge>
          )}
        </div>

        {/* RIGHT — Lock, Export, Runs */}
        <div className="flex items-center gap-1">
          {/* Lock status / actions */}
          <LockCluster canEdit={canEdit} editing={editing} lockOwner={lockOwner} onAcquireLock={onAcquireLock} onReleaseLock={onReleaseLock} />

          <Separator orientation="vertical" className="mx-2 h-6" />

          {/* Export Terraform */}
          <ToolTipButton ariaLabel="Export Terraform" label="Export Terraform" description="Generates Terraform from the current graph." onClick={onExportTerraform} disabled={exporting}>
            {exporting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Download className="h-4 w-4" />}
          </ToolTipButton>

          {/* Plan / Apply (workspace bound) */}
          {workspaceBound ? (
            <>
              <Separator orientation="vertical" className="mx-2 h-6" />
              <ToolTipButton ariaLabel="Plan" label="Plan" description="Create a new run and generate a plan." onClick={onPlan} disabled={!canEdit || viewOnly || runInProgress}>
                {runInProgress ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
              </ToolTipButton>
              <ToolTipButton ariaLabel="Apply" label="Apply" description="Apply the latest approved plan." onClick={onApply} disabled={!canEdit || viewOnly || runInProgress}>
                {runInProgress ? <Loader2 className="h-4 w-4 animate-spin" /> : <Cloud className="h-4 w-4" />}
              </ToolTipButton>
            </>
          ) : null}
        </div>
      </div>
    </TooltipProvider>
  );
}

/* ----------------------------- Subcomponents ----------------------------- */

function ToolTipButton({ ariaLabel, label, description, kbd, disabled, onClick, children }: { ariaLabel: string; label: string; description?: string; kbd?: string[]; disabled?: boolean; onClick?: () => void; children: React.ReactNode }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button type="button" variant="ghost" size="icon" aria-label={ariaLabel} disabled={disabled} onClick={onClick}>
          {children}
        </Button>
      </TooltipTrigger>
      <TooltipContent side="bottom" className="flex items-center gap-2">
        <div className="flex flex-col">
          <span className="text-sm font-medium">{label}</span>
          {description ? <span className="text-xs text-muted-foreground">{description}</span> : null}
        </div>
        {kbd && <Kbd keys={kbd} />}
      </TooltipContent>
    </Tooltip>
  );
}

function Kbd({ keys }: { keys: string[] }) {
  return (
    <div className="ml-1 flex items-center gap-1">
      {keys.map((k, i) => (
        <kbd key={`${k}-${i}`} className="rounded-md border bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
          {k}
        </kbd>
      ))}
    </div>
  );
}

function LockCluster({ canEdit, editing, lockOwner, onAcquireLock, onReleaseLock }: { canEdit: boolean; editing: boolean; lockOwner?: string | null; onAcquireLock?: () => void; onReleaseLock?: () => void }) {
  const lockedByOther = !editing && !!lockOwner;
  const viewOnlyBadge = lockedByOther ? (
    <Badge variant="outline" className="gap-1">
      <Lock className="h-3.5 w-3.5" />
      View-only · {lockOwner}
    </Badge>
  ) : (
    <Badge variant="outline" className="gap-1">
      <ShieldAlert className="h-3.5 w-3.5" />
      View-only
    </Badge>
  );

  if (!canEdit) {
    return viewOnlyBadge;
  }

  return (
    <div className="flex items-center gap-1">
      {editing ? (
        <>
          <Badge variant="secondary" className="gap-1 text-emerald-700 dark:text-emerald-300">
            <Unlock className="h-3.5 w-3.5" />
            Editing
          </Badge>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button type="button" variant="ghost" size="sm" onClick={onReleaseLock} aria-label="Release edit lock">
                Release
              </Button>
            </TooltipTrigger>
            <TooltipContent side="bottom">Relinquish the single-editor lock so others can edit.</TooltipContent>
          </Tooltip>
        </>
      ) : lockedByOther ? (
        viewOnlyBadge
      ) : (
        <Tooltip>
          <TooltipTrigger asChild>
            <Button type="button" variant="ghost" size="sm" onClick={onAcquireLock} aria-label="Request edit lock">
              <Lock className="mr-2 h-4 w-4" />
              Request edit
            </Button>
          </TooltipTrigger>
          <TooltipContent side="bottom">Acquire the single-editor lock to make changes.</TooltipContent>
        </Tooltip>
      )}
    </div>
  );
}
