"use client";

import ConnectCloudSheet from "@/components/connections/ConnectCloudSheet";
import ErrorState from "@/components/common/ErrorState";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useConnections } from "@/lib/features/connections/hooks";
import { useOrgDeployments } from "@/lib/features/deployments/orgHooks";
import { useHasDrawnDiagram } from "@/lib/features/diagram/hooks";
import { checklistStartsOpen, usePreferencesStore } from "@/lib/features/preferences/uiStore";
import { checklistLinks, checklistProgress, type StepId, type StepLink } from "@/lib/features/projects/checklist";
import { useProjects } from "@/lib/features/projects/hooks";
import { useMe } from "@/lib/features/user/hooks";
import { cn } from "@/lib/utils";
import { Check, ChevronDown, ChevronUp, X } from "lucide-react";
import Link from "next/link";
import { useEffect, useId, useMemo, useState } from "react";

type Props = {
  organizationId: string;
  // "editor": a small card docked in the canvas's bottom-left corner, folding
  // to a pill. "card": "Getting started" on Projects.
  variant: "editor" | "card";
  // In the editor: opens the editor's own Deploy view or code panel, on
  // every press (a URL that already says ?panel=code would not change).
  onOpenPanel?: (panel: "deploy" | "code") => void;
  // The blocks on the editor's canvas now, saved or not. Without it, "Draw
  // your setup" ticks once one of the organization's diagrams holds a block.
  blockCount?: number;
};

// FirstDeployChecklist is "Get to your first deploy": draw, connect a cloud,
// preview the code, deploy. Each step ticks from real state: blocks on the
// canvas (or in one of the organization's diagrams), a connection, a
// Terraform preview opened in this browser (or one that showed code), a live
// deployment. Connect opens the connection flow in a side sheet, starting on
// the cloud chosen at onboarding. It is open for new and console users and
// starts folded for Terraform users; folding and closing are remembered per
// organization. The editor's card and the card on Projects both go once the
// first deployment is live (projects/checklist.ts owns that rule), and stay
// gone: that moment closes it as the close button does.
export default function FirstDeployChecklist({ organizationId, variant, onOpenPanel, blockCount }: Props) {
  const meQ = useMe();
  const token = meQ.data?.token ?? null;
  const experience = usePreferencesStore((s) => s.experience);
  const firstCloud = usePreferencesStore((s) => s.firstCloud);
  const dismissed = usePreferencesStore((s) => !!s.checklistDismissed[organizationId]);
  const previewed = usePreferencesStore((s) => !!s.previewOpened[organizationId]);
  const lastDiagram = usePreferencesStore((s) => s.lastDiagram[organizationId] ?? null);
  const dismiss = usePreferencesStore((s) => s.dismissChecklist);
  const collapsed = usePreferencesStore((s) => s.checklistCollapsed[organizationId]);
  const setCollapsed = usePreferencesStore((s) => s.setChecklistCollapsed);
  const open = checklistStartsOpen(collapsed, experience);
  const setOpen = (o: boolean) => setCollapsed(organizationId, !o);
  const [connecting, setConnecting] = useState(false);
  const listId = useId();

  const connectionsQ = useConnections(token, organizationId);
  const deploymentsQ = useOrgDeployments(token, organizationId);
  // Only the card needs the organization's diagrams; the editor counts its canvas.
  const onCard = blockCount === undefined;
  const projectsQ = useProjects(onCard ? token : null, organizationId);
  const projectIds = useMemo(() => (projectsQ.data ?? []).map((p) => p.id), [projectsQ.data]);
  const drawnQ = useHasDrawnDiagram(onCard ? token : null, projectIds);
  const live = (deploymentsQ.data ?? []).some((d) => d.status === "live");
  const markPreviewOpened = usePreferencesStore((s) => s.markPreviewOpened);

  // The first live deployment finishes the checklist for good: a later
  // destroy or remove does not bring it back.
  useEffect(() => {
    if (live && !dismissed) dismiss(organizationId);
  }, [live, dismissed, dismiss, organizationId]);

  if (dismissed) return null;

  const loading = meQ.isLoading || connectionsQ.isLoading || deploymentsQ.isLoading || (onCard && (projectsQ.isLoading || drawnQ.isLoading));
  const failed =
    (connectionsQ.isError && !connectionsQ.data) ||
    (deploymentsQ.isError && !deploymentsQ.data) ||
    (onCard && ((projectsQ.isError && !projectsQ.data) || drawnQ.isError));

  const done: Record<StepId, boolean> = {
    draw: onCard ? drawnQ.drawn : blockCount > 0,
    connect: (connectionsQ.data ?? []).length > 0,
    preview: previewed,
    deploy: live,
  };
  const { count, next, finished } = checklistProgress(done);
  if (finished) return null;

  // In the editor a step opens its view in place; on Projects every step not
  // done has its own way there (projects/checklist.ts).
  const inEditor = variant === "editor" && !!onOpenPanel;
  const openPanel = (panel: "deploy" | "code") => onOpenPanel?.(panel);
  const links = checklistLinks(lastDiagram, projectsQ.data);
  const linkButton = (id: StepId, link: StepLink) => (
    <Button asChild size="xs" variant={next === id ? "primary" : "outline"}>
      <Link href={link.href}>{link.label}</Link>
    </Button>
  );

  const steps: { id: StepId; title: string; hint: string; action: React.ReactNode }[] = [
    {
      id: "draw",
      title: "Draw your setup",
      hint: inEditor ? "Click or drag a block from the left onto the canvas." : "Open a diagram, or make one blank or from an example.",
      action: inEditor ? null : linkButton("draw", links.draw),
    },
    {
      id: "connect",
      title: "Connect a cloud",
      hint: "A connection is a cloud account Clutter can build in.",
      action: (
        <Button size="xs" variant={next === "connect" ? "primary" : "outline"} onClick={() => setConnecting(true)}>
          Connect a cloud
        </Button>
      ),
    },
    {
      id: "preview",
      title: "Preview the code",
      hint: "See the Terraform Clutter writes before anything is built.",
      action: inEditor ? (
        <Button
          size="xs"
          variant={next === "preview" ? "primary" : "outline"}
          onClick={() => {
            // Opening the panel is the step, even on a cloud with no preview.
            markPreviewOpened(organizationId);
            openPanel("code");
          }}>
          Preview the code
        </Button>
      ) : (
        linkButton("preview", links.preview)
      ),
    },
    {
      id: "deploy",
      title: "Deploy",
      hint: "Build it in your cloud. You confirm first, and you can destroy it later.",
      action: inEditor ? (
        <Button size="xs" variant={next === "deploy" ? "primary" : "outline"} onClick={() => openPanel("deploy")}>
          Open Deploy
        </Button>
      ) : (
        linkButton("deploy", links.deploy)
      ),
    },
  ];

  const heading = variant === "card" ? "Getting started" : "Get to your first deploy";
  const compact = variant === "editor";
  const sheet = (
    <ConnectCloudSheet
      open={connecting}
      onOpenChange={setConnecting}
      organizationId={organizationId}
      preferredConnector={firstCloud ?? undefined}
      onConnected={() => setConnecting(false)}
    />
  );

  // Folded in the editor: one small pill that keeps the canvas clear.
  if (compact && !open) {
    return (
      <section aria-label={heading} data-testid="first-deploy-checklist" data-variant={variant} data-collapsed="">
        <button
          type="button"
          aria-expanded={false}
          aria-label={`${heading}: ${count} of 4 done. Show the steps.`}
          onClick={() => setOpen(true)}
          className="surface-raised text-foreground hover:bg-accent focus-visible:ring-ring/50 inline-flex h-8 items-center gap-2 rounded-full border pr-3 pl-2 text-xs font-medium transition-colors outline-none focus-visible:ring-[3px] animate-fade-in">
          <span className="bg-muted rounded-full px-1.5 tabular-nums">{count}/4</span>
          done
          <ChevronUp className="text-muted-foreground size-3.5" aria-hidden />
        </button>
        {sheet}
      </section>
    );
  }

  return (
    <section
      aria-label={heading}
      data-testid="first-deploy-checklist"
      data-variant={variant}
      className={cn(
        "text-card-foreground animate-fade-in",
        // In the editor it floats over the canvas, so it stays opaque, and
        // stops below the canvas's top-left controls.
        compact ? "surface-raised flex max-h-[calc(100dvh-11rem)] w-72 max-w-full flex-col rounded-xl border" : "surface-card w-full",
      )}>
      <div className={cn("flex shrink-0 items-center gap-2", compact ? "py-2 pr-2 pl-3" : "px-4 py-3")}>
        <button
          type="button"
          aria-expanded={open}
          aria-controls={listId}
          onClick={() => setOpen(!open)}
          className="focus-visible:ring-ring/50 -mx-1 flex min-w-0 flex-1 items-center gap-2 rounded-md px-1 text-left outline-none focus-visible:ring-[3px]">
          <span className="min-w-0 flex-1">
            <span className="block truncate text-sm font-semibold">{heading}</span>
            <span className="text-muted-foreground block text-xs">{count} of 4 done</span>
          </span>
          <ChevronDown className={cn("text-muted-foreground size-4 shrink-0 transition-transform", !open && "-rotate-90")} aria-hidden />
        </button>
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          aria-label={`Close ${heading.toLowerCase()}`}
          title="Close. It stays closed for this team in this browser."
          onClick={() => dismiss(organizationId)}>
          <X aria-hidden />
        </Button>
      </div>
      <div aria-hidden className={cn("bg-muted h-1 shrink-0 overflow-hidden rounded-full", compact ? "mx-3" : "mx-4")}>
        <div className="bg-primary h-full origin-left rounded-full transition-transform motion-slow" style={{ transform: `scaleX(${count / 4})` }} />
      </div>
      {open && (
        <div id={listId} className={compact ? "min-h-0 overflow-y-auto px-2 pt-1 pb-2" : "px-4 pt-2 pb-4"}>
          {loading ? (
            <div role="status" className="space-y-3 pt-2">
              <span className="sr-only">Checking your progress…</span>
              {[0, 1, 2, 3].map((i) => (
                <Skeleton key={i} className="h-9" />
              ))}
            </div>
          ) : (
            <ol className="space-y-1 pt-1">
              {failed && (
                <li className="px-2 py-1">
                  <ErrorState
                    variant="inline"
                    description="Clutter could not check every step."
                    onRetry={() => {
                      void connectionsQ.refetch();
                      void deploymentsQ.refetch();
                      if (onCard) {
                        void projectsQ.refetch();
                        drawnQ.refetch();
                      }
                    }}
                  />
                </li>
              )}
              {steps.map((s, i) => {
                const isDone = done[s.id];
                const isNext = s.id === next;
                return (
                  <li
                    key={s.id}
                    data-step={s.id}
                    data-done={isDone || undefined}
                    className={cn("rounded-lg px-2", compact ? "py-1.5" : "py-2", isNext && "bg-muted/60")}>
                    <div className="flex items-start gap-3">
                      <span
                        className={cn(
                          "mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full border text-[11px] font-semibold",
                          isDone ? "bg-success border-success text-success-foreground" : "text-muted-foreground border-border",
                        )}>
                        {isDone ? <Check className="size-3 animate-check" strokeWidth={3} aria-hidden /> : i + 1}
                      </span>
                      <span className="min-w-0 flex-1">
                        <span className={cn("block text-sm font-medium", isDone && "text-muted-foreground line-through decoration-1")}>
                          {s.title}
                          <span className="sr-only">{isDone ? ": done" : ": to do"}</span>
                        </span>
                        {isNext && <span className="text-muted-foreground mt-0.5 block text-xs">{s.hint}</span>}
                        {isNext && s.action && <span className="mt-2 flex">{s.action}</span>}
                      </span>
                      {!compact && !isNext && !isDone && s.action && <span className="shrink-0">{s.action}</span>}
                    </div>
                  </li>
                );
              })}
            </ol>
          )}
        </div>
      )}
      {sheet}
    </section>
  );
}
