"use client";

import FirstLiveStack from "@/components/brand/FirstLiveStack";
import CloudLogo from "@/components/common/CloudLogo";
import StatusDot from "@/components/common/StatusDot";
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
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { toast } from "@/components/ui/sonner";
import { AdapterApiError, adapterErrorMessage } from "@/lib/adapterApi";
import { cloudLookFor } from "@/lib/features/catalog/display";
import { cleanValues, cloudOffers, defaultValues, entryFor, settingsFor, valuesErrors } from "@/lib/features/catalog/forms";
import type { CatalogEntry } from "@/lib/features/catalog/types";
import type { Connection } from "@/lib/features/connections/types";
import { ACTION_LABEL, actionReasons, allLeftOut, canOfferRemove, type RowAction } from "@/lib/features/deployments/actions";
import { compatibilityLine, nextExclusions, splitReadyLine, unsupportedBlocks, type BlockRef, type CompatibilityLine } from "@/lib/features/deployments/compatibility";
import { costLines, deployLine, destroyLine } from "@/lib/features/deployments/confirm";
import { useCompatibility, useRemoveDeployment, useRunLogs, useStartRun, useUpdateDeployment } from "@/lib/features/deployments/hooks";
import {
  cloudBlocks,
  cloudsLine,
  crossNotes,
  deploymentTitle,
  needsTwoClouds,
  orderedClouds,
  partLines,
  placeBlock,
  placementReason,
  placementSuggestions,
  splitOnly,
} from "@/lib/features/deployments/placements";
import { DEPLOYMENT_STATE, deploymentState, mergeRuns, primaryAction, primaryRun, runEndToast, runLabel } from "@/lib/features/deployments/runState";
import { changesNotDeployed, isActive, type DeploymentListItem, type Run, type RunAction } from "@/lib/features/deployments/types";
import { useRunHistoryStore } from "@/lib/features/deployments/uiStore";
import { usePreferencesStore } from "@/lib/features/preferences/uiStore";
import { cn } from "@/lib/utils";
import { CheckCircle2, ChevronDown, ChevronRight, Code2, History, KeyRound, Loader2, Play, Rocket, ScrollText, Trash2, TriangleAlert, X } from "lucide-react";
import { useEffect, useId, useMemo, useRef, useState } from "react";
import BlockPlacement from "./BlockPlacement";
import RunLog, { RunHeadline } from "./RunLog";
import SettingsForm from "./SettingsForm";
import WhatWasBuilt from "./WhatWasBuilt";

type Props = {
  token: string | null;
  organizationId: string;
  diagramId: string;
  item: DeploymentListItem;
  entry: CatalogEntry | null;
  // Every cloud this server offers and the organization's connections, for
  // placing blocks on other clouds.
  catalog: CatalogEntry[];
  connections: Connection[];
  blocks: BlockRef[];
  blockTitle: (kind: string) => string;
  // The servers the saved diagram makes a website (results.servesWeb).
  webServers: ReadonlySet<string>;
  // Set while the canvas has unsaved changes: checks and runs use the saved diagram.
  blockedReason: string | null;
  // The deployment the canvas shows (live markers, the code panel).
  selected: boolean;
  // Whether the card starts unfolded: the only deployment, or the selected one.
  defaultOpen: boolean;
  onSelect: () => void;
  // Opens the code panel on this deployment.
  onPreview: () => void;
};

const PART_TONE: Record<string, string> = {
  completed: "text-success",
  failed: "text-danger",
  rolled_back: "text-warning",
  skipped: "text-muted-foreground",
};

const RUN_TONE: Record<Run["status"], "info" | "primary" | "success" | "danger"> = {
  queued: "info",
  running: "primary",
  completed: "success",
  failed: "danger",
};

const when = (r: Run) => new Date(r.finished_at ?? r.started_at ?? r.created_at).toLocaleString();

// One deployment of the diagram: where it stands in one named state with the
// one action that state calls for, what the cloud can build, where each block
// runs, its settings, its runs and their logs, and what the last deploy built.
// Destroying and removing sit apart, each behind a confirmation.
export default function DeploymentRow({
  token,
  organizationId,
  diagramId,
  item,
  entry,
  catalog,
  connections,
  blocks,
  blockTitle,
  webServers,
  blockedReason,
  selected,
  defaultOpen,
  onSelect,
  onPreview,
}: Props) {
  const cloud = entry?.manifest.title ?? item.connector;
  const look = cloudLookFor(entry ? [entry] : [], item.connector);
  const settingsSpec = entry?.manifest.settings ?? [];
  const placements = item.placements ?? [];

  const compatQ = useCompatibility(token, organizationId, item.id);
  const updateM = useUpdateDeployment(token);
  // While a change to the exclusions is saving, show the change.
  const exclusions = (updateM.isPending ? updateM.variables?.data.exclusions : undefined) ?? item.exclusions ?? [];
  const startM = useStartRun(token);
  const removeM = useRemoveDeployment(token);
  const markPreviewOpened = usePreferencesStore((s) => s.markPreviewOpened);
  const offId = useId();
  const bodyId = useId();
  const [open, setOpen] = useState(defaultOpen);

  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [settings, setSettings] = useState<Record<string, unknown>>(() => ({ ...defaultValues(settingsSpec), ...(item.settings ?? {}) }));
  const [logOpen, setLogOpen] = useState(false);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [historyLog, setHistoryLog] = useState<string | null>(null);
  const [confirm, setConfirm] = useState<"apply" | "destroy" | "remove" | null>(null);
  // The last successful apply's log, for what a destroy removes. It is the
  // query "What was built" already reads, so it is fetched once.
  const lastApplyLogQ = useRunLogs(token, organizationId, confirm === "destroy" ? (item.last_apply ?? null) : null);

  // The runs this page has seen, with the two B returns for the deployment.
  const record = useRunHistoryStore((s) => s.record);
  const seen = useRunHistoryStore((s) => s.byDeployment[item.id]);
  useEffect(() => {
    record(item.id, [item.latest_run, item.last_apply]);
  }, [record, item.id, item.latest_run, item.last_apply]);
  const history = useMemo(() => mergeRuns(seen ?? [], [item.latest_run, item.last_apply]), [seen, item.latest_run, item.last_apply]);

  const run = item.latest_run;
  const running = isActive(run) || startM.isPending;
  const state = deploymentState(item);
  const stateInfo = DEPLOYMENT_STATE[state];
  const primary = primaryAction(item);
  const report = compatQ.data;
  const titleOf = (connector: string) => entryFor(catalog, connector)?.manifest.title ?? connector;
  const offers = (connector: string, kind: string) => cloudOffers(catalog, connector, kind);
  const title = deploymentTitle(item.connection_name, placements);
  const where = cloudsLine({ title: cloud, region: item.region }, placements, titleOf);
  // The same clouds without regions, for the lines that say what a run does.
  const builtIn = cloudsLine({ title: cloud, region: "" }, placements.map((p) => ({ ...p, region: "" })), titleOf);

  // The blocks each cloud runs: a split deployment has a line per cloud, in
  // the order a run takes them, each about its own blocks.
  const clouds = report ? orderedClouds(report) : [];
  const ownBlocks = cloudBlocks(item.connection_id, placements, blocks, item.connection_id);
  const lines: CompatibilityLine[] = !report
    ? []
    : clouds.length > 0
      ? clouds.map((c) =>
          compatibilityLine(c.connection_name, c, cloudBlocks(item.connection_id, placements, blocks, c.connection_id), exclusions, blockTitle, true),
        )
      : [compatibilityLine(cloud, report, blocks, exclusions, blockTitle)];
  // A split deployment that every cloud can build says so once, for all of them.
  const splitReady = splitReadyLine(
    lines,
    clouds.map((c) => c.connection_name),
    exclusions.filter((id) => blocks.some((b) => b.id === id)).length,
  );
  const extra = report && clouds.length > 0 ? splitOnly(report) : { problems: [], notes: [] };
  const cross = report ? crossNotes(report) : { keys: [], addresses: [] };
  const byId = new Map(blocks.map((b) => [b.id, b.name]));
  const ready = !!report?.ready;

  // Blocks the own cloud cannot build, and the connection that can run each.
  const unsupported = entry ? unsupportedBlocks(entry.manifest, ownBlocks) : [];
  const moveReason = placementReason(item.status, running || updateM.isPending);
  const suggestions = new Map(placementSuggestions(unsupported, item.connection_id, connections, offers).map((s) => [s.block.id, s.connection]));

  // Settings are about some blocks (a function runtime about functions); only
  // those about this cloud's blocks show.
  const shownSettings = settingsFor(settingsSpec, new Set(ownBlocks.map((b) => b.kind)));
  const settingsErrors = valuesErrors(shownSettings, settings);

  const offersRemove = canOfferRemove(item.status);

  const reasons = actionReasons({
    status: item.status,
    running,
    saving: updateM.isPending || removeM.isPending,
    unsaved: blockedReason,
    checking: compatQ.isLoading,
    checkFailed: compatQ.isError,
    ready,
    allLeftOut: allLeftOut(
      blocks.map((b) => b.id),
      exclusions,
    ),
    noBlocks: blocks.length === 0,
  });
  // "Up to date" is off for its own reason: the cloud holds the saved diagram.
  // "Destroy again" is a destroy, so the destroy's own reasons turn it off.
  const primaryRunAction = primaryRun(primary.kind);
  const primaryReason =
    primary.kind === "up_to_date"
      ? "Your cloud already has the saved diagram and settings. Change and save them to update."
      : primaryRunAction === "destroy"
        ? reasons.destroy
        : primaryRunAction === "apply"
          ? reasons.apply
          : null;

  // The run whose "started" toast is up: once the run ends, the same toast
  // says how it ended, so a finished run never reads as just started.
  const toastedRun = useRef<string | null>(null);
  useEffect(() => {
    if (!run || run.id !== toastedRun.current) return;
    const end = runEndToast(run, title);
    if (!end) return;
    toastedRun.current = null;
    if (end.ok) toast.success(end.text, { id: run.id });
    else toast.error(end.text, { id: run.id });
  }, [run, title]);

  const start = (action: RunAction) => {
    setConfirm(null);
    startM.mutate(
      { organizationId, diagramId, deploymentId: item.id, action },
      {
        onSuccess: (started) => {
          toastedRun.current = started.id;
          toast(`${ACTION_LABEL[action === "apply" ? "apply" : action]} started on ${title}`, { id: started.id });
        },
      },
    );
    if (action !== "plan") setLogOpen(false);
  };

  const remove = () => {
    setConfirm(null);
    removeM.mutate({ organizationId, diagramId, deploymentId: item.id }, { onSuccess: () => toast.success(`${title} removed from the diagram`) });
  };

  const toggleExclusion = (id: string, exclude: boolean) =>
    updateM.mutate({ organizationId, diagramId, deploymentId: item.id, data: { exclusions: nextExclusions(exclusions, blocks, id, exclude) } });

  const runOn = (blockId: string, connectionId: string) =>
    updateM.mutate({
      organizationId,
      diagramId,
      deploymentId: item.id,
      data: placeBlock(item.connection_id, placements, exclusions, blocks, blockId, connectionId),
    });

  // What a press of each button does, said on hover instead of in the panel.
  const help: Record<RowAction, string> = {
    preview: "See the Terraform code Clutter writes for this deployment.",
    plan: "Shows what would change. Nothing is built.",
    apply: `Builds it in ${builtIn}.`,
    destroy: "Removes everything this deployment built in your cloud.",
    remove: "Takes this deployment off the diagram. Nothing in your cloud changes.",
  };
  const off = (action: RowAction | "primary", reason: string | null) =>
    reason
      ? { "aria-disabled": true, "aria-describedby": `${offId}-${action}`, title: reason }
      : { title: action === "primary" ? (primaryRunAction === "destroy" ? help.destroy : help.apply) : help[action] };

  const onPrimary = () => {
    if (primary.kind === "log") return setLogOpen(true);
    if (!primaryReason && primaryRunAction) setConfirm(primaryRunAction);
  };
  const primaryIcon =
    primary.kind === "log" ? (
      <ScrollText aria-hidden />
    ) : primary.kind === "up_to_date" ? (
      <CheckCircle2 aria-hidden />
    ) : primaryRunAction === "destroy" ? (
      <Trash2 aria-hidden />
    ) : (
      <Rocket aria-hidden />
    );
  const primaryLabel = primary.kind === "log" ? "Watch the log" : primary.label;

  return (
    <li
      data-testid="deployment-row"
      data-connector={item.connector}
      data-state={state}
      data-selected={selected || undefined}
      onClickCapture={onSelect}
      onFocusCapture={onSelect}
      className={cn("surface-card text-card-foreground space-y-4 p-4 transition-shadow", selected && "ring-primary/30 ring-2")}>
      {/* Where it stands. The whole header folds the card. */}
      <button
        type="button"
        data-testid="deployment-toggle"
        aria-expanded={open}
        aria-controls={bodyId}
        aria-label={`${open ? "Fold" : "Unfold"} ${title}`}
        onClick={() => setOpen((o) => !o)}
        className="-m-1 flex w-[calc(100%+0.5rem)] items-start justify-between gap-3 rounded-md p-1 text-left outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50">
        <div className="flex min-w-0 items-center gap-3">
          <CloudLogo connector={look.connector} shortName={look.shortName} src={look.logoSrc} size={28} />
          <div className="min-w-0">
            <div data-testid="deployment-title" className="text-sm font-semibold break-words">
              {title}
            </div>
            <div data-testid="deployment-clouds" className="text-muted-foreground truncate text-xs" title={where}>
              {where}
            </div>
          </div>
        </div>
        <div className="flex shrink-0 flex-col items-end gap-1">
          <div className="relative">
            {/* Beside the badge, out of the layout: nothing moves while it plays. */}
            <FirstLiveStack live={state === "live"} className="absolute top-1/2 right-full mr-1.5 -translate-y-1/2" />
            <Badge variant={stateInfo.tone === "primary" ? "primary" : stateInfo.tone} data-testid="deployment-state" data-state={state}>
              <StatusDot tone={stateInfo.tone} pulse={stateInfo.running} />
              {stateInfo.label}
            </Badge>
          </div>
          {changesNotDeployed(item) && (
            <Badge variant="warning" data-testid="changes-not-deployed" title="The saved diagram or this deployment's settings have changes the last deploy did not build. Update to build them.">
              Changes not deployed yet
            </Badge>
          )}
        </div>
        <ChevronDown aria-hidden className={cn("text-muted-foreground mt-1 size-4 shrink-0 transition-transform motion-reduce:transition-none", !open && "-rotate-90")} />
      </button>
      {open && (
        <div id={bodyId} className="space-y-4">
      {/* pl-10: under the title, past the 28px logo and its 12px gap. */}
      <p className="text-muted-foreground -mt-1 pl-10 text-xs">{stateInfo.hint}</p>

      {/* The one thing to do now, then the quieter ones. */}
      <div className="flex flex-wrap items-center gap-2">
        <Button
          type="button"
          size="sm"
          variant={primary.kind === "up_to_date" ? "outline" : primaryRunAction === "destroy" ? "destructive" : "primary"}
          data-deployment-primary={item.id}
          {...off("primary", primaryReason)}
          onClick={primaryReason ? undefined : onPrimary}
          className="aria-disabled:cursor-not-allowed aria-disabled:opacity-50 aria-disabled:hover:translate-y-0">
          {startM.isPending ? <Loader2 className="animate-spin" aria-hidden /> : primaryIcon}
          {primaryLabel}
        </Button>
        <Button
          type="button"
          size="sm"
          variant="outline"
          {...off("plan", reasons.plan)}
          onClick={reasons.plan ? undefined : () => start("plan")}
          className="aria-disabled:cursor-not-allowed aria-disabled:opacity-50 aria-disabled:hover:translate-y-0">
          <Play aria-hidden />
          {ACTION_LABEL.plan}
        </Button>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          {...off("preview", reasons.preview && !blockedReason ? reasons.preview : null)}
          onClick={
            reasons.preview && !blockedReason
              ? undefined
              : () => {
                  // Opening the preview ticks the checklist's step, even on a
                  // cloud that offers no Terraform preview.
                  markPreviewOpened(organizationId);
                  onPreview();
                }
          }
          className="aria-disabled:cursor-not-allowed aria-disabled:opacity-50">
          <Code2 aria-hidden />
          Preview code
        </Button>
      </div>
      {/* Why each button is off, for screen readers; sighted users see it on hover. */}
      {(
        [
          ["primary", primaryReason],
          ["plan", reasons.plan],
          ["preview", reasons.preview && !blockedReason ? reasons.preview : null],
          ["destroy", reasons.destroy],
          ["remove", reasons.remove],
        ] as const
      ).map(([a, r]) =>
        r ? (
          <span key={a} id={`${offId}-${a}`} className="sr-only">
            {a === "primary" ? primaryLabel : ACTION_LABEL[a]} is off. {r}
          </span>
        ) : null,
      )}
      {blockedReason && <p className="text-warning -mt-2 text-xs">{blockedReason}</p>}
      {startM.isError && (
        <div role="alert" className="text-danger text-xs">
          <p>{adapterErrorMessage(startM.error)}</p>
          {startM.error instanceof AdapterApiError &&
            startM.error.problems.map((p, i) => (
              <p key={i} className="pl-3">
                {p.message}
              </p>
            ))}
        </div>
      )}

      {/* The latest run */}
      {run ? (
        <div className="space-y-2 rounded-lg border p-3">
          <div className="flex items-center justify-between gap-2">
            <p data-testid="run-status" data-status={run.status} className="flex min-w-0 items-center gap-2 text-xs">
              <StatusDot tone={RUN_TONE[run.status]} pulse={isActive(run)} />
              <span className="font-medium">{runLabel(run)}</span>
              {run.finished_at && <span className="text-muted-foreground truncate">at {new Date(run.finished_at).toLocaleTimeString()}</span>}
            </p>
            <Button type="button" variant="ghost" size="xs" aria-expanded={logOpen} onClick={() => setLogOpen((o) => !o)}>
              <ScrollText aria-hidden />
              {logOpen ? "Hide log" : "Show log"}
            </Button>
          </div>
          {run.parts && run.parts.length > 0 && (
            <ol data-testid="run-parts" aria-label="Each cloud of this run" className="space-y-0.5 pl-4 text-xs">
              {partLines(run.parts).map((p) => (
                <li key={p.connectionId} className={cn("flex items-center gap-1.5", PART_TONE[p.status])}>
                  {(p.status === "running" || p.status === "queued") && <Loader2 className="size-3 animate-spin" aria-hidden />}
                  {p.text}
                </li>
              ))}
            </ol>
          )}
          {!isActive(run) && <RunHeadline token={token} organizationId={organizationId} run={run} />}
          {logOpen && <RunLog token={token} organizationId={organizationId} run={run} title={`${runLabel(run)} on ${title}`} blockTitle={blockTitle} />}
        </div>
      ) : (
        <p className="text-muted-foreground text-xs">No runs yet. Plan shows what would change; Deploy builds it.</p>
      )}

      {/* What the last successful apply built, while the deployment still holds it. */}
      {item.last_apply && (item.status === "live" || item.status === "failed") && (
        <WhatWasBuilt
          token={token}
          organizationId={organizationId}
          item={item}
          run={item.last_apply}
          catalog={catalog}
          blocks={blocks}
          blockTitle={blockTitle}
          webServers={webServers}
        />
      )}

      {/* Compatibility */}
      <div data-testid="compatibility" className="space-y-1.5">
        {compatQ.isLoading ? (
          <div role="status" className="text-muted-foreground flex items-center gap-2 text-xs">
            <Loader2 className="size-3.5 animate-spin" aria-hidden />
            Checking what {cloud} can build…
          </div>
        ) : compatQ.isError ? (
          <div className="flex items-center gap-2">
            <p className="text-danger text-xs">Could not check what {cloud} can build. {adapterErrorMessage(compatQ.error)}</p>
            <Button type="button" variant="outline" size="xs" onClick={() => void compatQ.refetch()}>
              Retry
            </Button>
          </div>
        ) : report ? (
          <>
            {splitReady && (
              <p className="text-success flex items-start gap-1.5 text-xs">
                <CheckCircle2 className="mt-px size-3.5 shrink-0" aria-hidden />
                {splitReady}
              </p>
            )}
            {lines.map((line) => (
              <div key={line.summary} className="space-y-1.5">
                {!splitReady && (
                  <p className={cn("flex items-start gap-1.5 text-xs", line.ready ? "text-success" : "text-warning")}>
                    {line.ready ? <CheckCircle2 className="mt-px size-3.5 shrink-0" aria-hidden /> : <TriangleAlert className="mt-px size-3.5 shrink-0" aria-hidden />}
                    {line.summary}
                  </p>
                )}
                {line.choices.map((c) => {
                  const there = !c.supported && !moveReason ? suggestions.get(c.id) : undefined;
                  return (
                    <div key={c.id} className="space-y-1 pl-5">
                      {there && (
                        <Button
                          type="button"
                          size="xs"
                          data-testid="run-on-suggestion"
                          disabled={updateM.isPending || running}
                          onClick={() => runOn(c.id, there.id)}>
                          Run {c.name} on {there.name}
                        </Button>
                      )}
                      <label className="flex min-h-6 items-center gap-2 text-xs">
                        <input
                          type="checkbox"
                          className="accent-primary size-4"
                          checked={c.excluded}
                          disabled={updateM.isPending || running}
                          onChange={(e) => toggleExclusion(c.id, e.target.checked)}
                        />
                        <span>
                          Leave {c.name} ({c.kindTitle}) out of this deployment
                          {!c.supported && !there && <span className="text-warning"> (not supported here)</span>}
                        </span>
                      </label>
                    </div>
                  );
                })}
                {line.problems.map((p) => (
                  <p key={p} className="text-warning pl-5 text-xs">
                    {p}
                  </p>
                ))}
                {line.notes.map((n) => (
                  <p key={n} className="text-muted-foreground pl-5 text-xs">
                    {n}
                  </p>
                ))}
              </div>
            ))}
            {extra.problems.map((p) => (
              <p key={p.code + p.message} className="text-warning flex items-start gap-1.5 text-xs">
                <TriangleAlert className="mt-px size-3.5 shrink-0" aria-hidden />
                {p.edge_source && p.edge_target ? `${byId.get(p.edge_source) ?? p.edge_source} to ${byId.get(p.edge_target) ?? p.edge_target}: ` : ""}
                {p.message}
              </p>
            ))}
            {extra.notes.map((n) => (
              <p key={n.code + n.message} className="text-muted-foreground pl-5 text-xs">
                {n.message}
              </p>
            ))}
            {cross.keys.map((k) => (
              <p key={k} data-testid="scoped-key-note" className="text-info flex items-start gap-1.5 text-xs">
                <KeyRound className="mt-px size-3.5 shrink-0" aria-hidden />
                {k}
              </p>
            ))}
            {cross.addresses.map((a) => (
              <p key={a} className="text-muted-foreground pl-5 text-xs">
                {a}
              </p>
            ))}
          </>
        ) : null}
        {updateM.isError && <p className="text-danger text-xs">{adapterErrorMessage(updateM.error)}</p>}
      </div>

      {/* Where each block runs */}
      <BlockPlacement
        token={token}
        organizationId={organizationId}
        diagramId={diagramId}
        item={item}
        catalog={catalog}
        connections={connections}
        blocks={blocks}
        blockTitle={blockTitle}
        running={running}
        defaultOpen={needsTwoClouds(placements, unsupported.length)}
      />

      {/* Advanced: this cloud's settings for the blocks it runs. */}
      {shownSettings.length > 0 && (
        <div>
          <button
            type="button"
            aria-expanded={advancedOpen}
            onClick={() => setAdvancedOpen((o) => !o)}
            className="text-muted-foreground hover:text-foreground flex min-h-6 items-center gap-1 text-xs font-medium">
            {advancedOpen ? <ChevronDown className="size-3.5" aria-hidden /> : <ChevronRight className="size-3.5" aria-hidden />}
            Advanced
          </button>
          {advancedOpen && (
            <div className="mt-2 space-y-3 pl-1">
              <p className="text-muted-foreground text-[11px]">{cloud} settings</p>
              <SettingsForm
                token={token}
                organizationId={organizationId}
                connectionId={item.connection_id}
                idPrefix={`deployment-${item.id}`}
                spec={shownSettings}
                values={settings}
                disabled={running}
                onChange={(name, v) => setSettings((s) => ({ ...s, [name]: v }))}
              />
              <Button
                type="button"
                variant="secondary"
                size="xs"
                disabled={updateM.isPending || running || settingsErrors.length > 0}
                onClick={() =>
                  updateM.mutate(
                    { organizationId, diagramId, deploymentId: item.id, data: { settings: cleanValues(settingsSpec, settings) } },
                    { onSuccess: () => toast.success("Settings saved") },
                  )
                }>
                {updateM.isPending ? "Saving…" : "Save settings"}
              </Button>
            </div>
          )}
        </div>
      )}

      {/* Run history */}
      {history.length > 0 && (
        <div>
          <button
            type="button"
            aria-expanded={historyOpen}
            onClick={() => setHistoryOpen((o) => !o)}
            className="text-muted-foreground hover:text-foreground flex min-h-6 items-center gap-1 text-xs font-medium">
            {historyOpen ? <ChevronDown className="size-3.5" aria-hidden /> : <ChevronRight className="size-3.5" aria-hidden />}
            <History className="size-3.5" aria-hidden />
            Recent runs ({history.length})
          </button>
          {historyOpen && (
            <div className="mt-2 space-y-1.5">
              <ol data-testid="run-history" className="divide-y rounded-lg border">
                {history.map((r) => (
                  <li key={r.id} data-testid="run-history-item" className="space-y-2 px-3 py-2">
                    <div className="flex items-center gap-2 text-xs">
                      <StatusDot tone={RUN_TONE[r.status]} pulse={isActive(r)} />
                      <span className="font-medium">{runLabel(r)}</span>
                      <span className="text-muted-foreground min-w-0 flex-1 truncate">{when(r)}</span>
                      <Button
                        type="button"
                        variant="ghost"
                        size="xs"
                        aria-expanded={historyLog === r.id}
                        aria-label={`${historyLog === r.id ? "Hide" : "Show"} the log of ${runLabel(r)}, ${when(r)}`}
                        onClick={() => setHistoryLog((cur) => (cur === r.id ? null : r.id))}>
                        Log
                      </Button>
                    </div>
                    {r.failure && <p className="text-danger text-xs">{r.failure}</p>}
                    {historyLog === r.id && (
                      <RunLog token={token} organizationId={organizationId} run={r} title={`${runLabel(r)} on ${title}`} blockTitle={blockTitle} testId="run-history-log" />
                    )}
                  </li>
                ))}
              </ol>
              <p className="text-muted-foreground text-[11px]">The latest run, the last successful deploy, and the runs since you opened this page.</p>
            </div>
          )}
        </div>
      )}

      {/* Taking things away sits apart from building them. */}
      <div data-testid="remove-actions" className="flex flex-wrap items-center gap-2 border-t pt-3">
        {!offersRemove && (
          <Button
            type="button"
            variant="outline"
            size="sm"
            {...off("destroy", reasons.destroy)}
            onClick={reasons.destroy ? undefined : () => setConfirm("destroy")}
            className="text-danger hover:text-danger aria-disabled:cursor-not-allowed aria-disabled:opacity-50">
            <Trash2 aria-hidden />
            {ACTION_LABEL.destroy}
          </Button>
        )}
        <Button
          type="button"
          variant="ghost"
          size="sm"
          {...off("remove", reasons.remove)}
          onClick={reasons.remove ? undefined : () => setConfirm("remove")}
          className="aria-disabled:cursor-not-allowed aria-disabled:opacity-50">
          {removeM.isPending ? <Loader2 className="animate-spin" aria-hidden /> : <X aria-hidden />}
          {removeM.isPending ? "Removing…" : ACTION_LABEL.remove}
        </Button>
      </div>
      {removeM.isError && (
        <p role="alert" className="text-danger text-xs">
          {adapterErrorMessage(removeM.error)}
        </p>
      )}
        </div>
      )}

      {/* Deploy asks first: what each cloud builds and what it costs beyond the blocks. */}
      <Dialog open={confirm === "apply"} onOpenChange={(o) => setConfirm(o ? "apply" : null)}>
        <DialogContent data-testid="deploy-confirm">
          <DialogHeader>
            <DialogTitle>
              {primary.kind === "update" ? "Update" : "Deploy"} {title}?
            </DialogTitle>
            <DialogDescription>{deployLine({ id: item.connection_id, name: item.connection_name }, placements, blocks, exclusions, blockTitle)}</DialogDescription>
          </DialogHeader>
          {costLines(report).map((c) => (
            <p key={c} data-testid="cost-line" className="border-warning/30 bg-warning/10 text-warning rounded-md border px-3 py-2 text-sm">
              {c}
            </p>
          ))}
          <p className="text-muted-foreground text-xs">It runs in your own cloud account. You can destroy it later.</p>
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="outline">
                Cancel
              </Button>
            </DialogClose>
            <Button type="button" onClick={() => start("apply")} autoFocus>
              <Rocket aria-hidden />
              {primary.kind === "update" ? "Update" : "Deploy"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog open={confirm === "destroy"} onOpenChange={(o) => setConfirm(o ? "destroy" : null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Destroy {title}?</AlertDialogTitle>
            <AlertDialogDescription>Destroy removes everything this deployment built in {builtIn}. This cannot be undone.</AlertDialogDescription>
          </AlertDialogHeader>
          {/* What goes, as the deploy confirmation says what it builds. */}
          <p data-testid="destroy-list" className="text-sm">
            {destroyLine({ id: item.connection_id, name: item.connection_name }, placements, blocks, exclusions, blockTitle, lastApplyLogQ.data)}
          </p>
          {changesNotDeployed(item) && (
            <p className="text-muted-foreground text-xs">The diagram or its settings changed since the last deploy. Destroy removes what that deploy built.</p>
          )}
          <AlertDialogFooter>
            <AlertDialogCancel>Keep it</AlertDialogCancel>
            <AlertDialogAction onClick={() => start("destroy")}>Destroy for good</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={confirm === "remove"} onOpenChange={(o) => setConfirm(o ? "remove" : null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove {title} from the diagram?</AlertDialogTitle>
            <AlertDialogDescription>Nothing in your cloud changes. The connection is free again for a new deployment of this diagram.</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Keep it</AlertDialogCancel>
            <AlertDialogAction onClick={remove}>Remove it</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </li>
  );
}
