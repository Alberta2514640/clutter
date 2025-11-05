"use client";

import * as React from "react";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription, SheetFooter, SheetClose } from "@/components/ui/sheet";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Play, Cloud, Loader2, CheckCircle2, XCircle, Terminal, FileText, Clock, Calendar } from "lucide-react";
import { cn } from "@/lib/utils";

// ---------------------------------------------------------------------

export type RunStatus = "planning" | "applying" | "succeeded" | "failed" | "idle";

export interface RunInfo {
  id: string;
  type: "plan" | "apply";
  startedAt?: string;
  finishedAt?: string;
  triggeredBy?: string;
  status: RunStatus;
  log?: string[];
}

interface RunDrawerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  run?: RunInfo | null;
  onRerun?: () => void;
  onClose?: () => void;
}

// ---------------------------------------------------------------------

export default function RunDrawer({ open, onOpenChange, run, onRerun, onClose }: RunDrawerProps) {
  if (!run) return null;

  const statusColor = run.status === "succeeded" ? "text-emerald-600" : run.status === "failed" ? "text-red-600" : run.status === "planning" ? "text-amber-600" : run.status === "applying" ? "text-sky-600" : "text-muted-foreground";

  const statusIcon =
    run.status === "planning" || run.status === "applying" ? (
      <Loader2 className="h-4 w-4 animate-spin" />
    ) : run.status === "succeeded" ? (
      <CheckCircle2 className="h-4 w-4" />
    ) : run.status === "failed" ? (
      <XCircle className="h-4 w-4" />
    ) : (
      <FileText className="h-4 w-4" />
    );

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="flex w-[480px] flex-col p-0">
        <SheetHeader className="p-4 pb-2">
          <SheetTitle className="flex items-center gap-2">
            {run.type === "plan" ? <Play className="h-5 w-5 text-sky-600" /> : <Cloud className="h-5 w-5 text-emerald-600" />}
            {run.type === "plan" ? "Terraform Plan" : "Terraform Apply"}
          </SheetTitle>
          <SheetDescription>
            Run ID: <span className="font-mono">{run.id}</span>
          </SheetDescription>
        </SheetHeader>

        <Separator />

        <div className="flex flex-col gap-2 p-4">
          <div className="flex items-center justify-between">
            <span className="text-sm text-muted-foreground">Status</span>
            <Badge variant="outline" className={cn("flex items-center gap-1 capitalize", statusColor)}>
              {statusIcon}
              {run.status}
            </Badge>
          </div>

          <div className="flex items-center justify-between text-sm text-muted-foreground">
            <span className="flex items-center gap-1">
              <Calendar className="h-4 w-4" /> Started
            </span>
            <span>{formatDate(run.startedAt)}</span>
          </div>

          <div className="flex items-center justify-between text-sm text-muted-foreground">
            <span className="flex items-center gap-1">
              <Clock className="h-4 w-4" /> Finished
            </span>
            <span>{formatDate(run.finishedAt) || "—"}</span>
          </div>

          <div className="flex items-center justify-between text-sm text-muted-foreground">
            <span>Triggered by</span>
            <span className="font-medium">{run.triggeredBy || "system"}</span>
          </div>
        </div>

        <Separator />

        <div className="flex-1 overflow-hidden">
          <ScrollArea className="h-full p-4">
            <div className="flex items-center gap-2 mb-2 text-muted-foreground">
              <Terminal className="h-4 w-4" />
              <span className="text-sm font-medium">Logs</span>
            </div>
            <div className="rounded-md bg-muted p-2 font-mono text-xs text-muted-foreground">
              {run.log?.length ? (
                run.log.map((line, i) => (
                  <div key={i} className="whitespace-pre-wrap">
                    {line}
                  </div>
                ))
              ) : (
                <div className="text-center text-sm text-muted-foreground">No logs yet.</div>
              )}
            </div>
          </ScrollArea>
        </div>

        <SheetFooter className="border-t p-4 flex justify-between">
          {run.status === "failed" && (
            <Button variant="outline" onClick={onRerun}>
              Retry
            </Button>
          )}
          <SheetClose asChild>
            <Button variant="default" onClick={onClose}>
              Close
            </Button>
          </SheetClose>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

// ---------------------------------------------------------------------

function formatDate(date?: string) {
  if (!date) return "";
  try {
    return new Date(date).toLocaleString();
  } catch {
    return date;
  }
}
