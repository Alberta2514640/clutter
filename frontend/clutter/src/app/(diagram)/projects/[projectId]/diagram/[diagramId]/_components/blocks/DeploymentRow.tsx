"use client";

import CloudMark from "@/components/clouds/CloudMark";
import { AdapterApiError, adapterErrorMessage } from "@/lib/adapterApi";
import { cloudLookFor } from "@/lib/features/catalog/display";
import { cleanValues, defaultValues, manifestOffers, settingsFor, valuesErrors } from "@/lib/features/catalog/forms";
import type { CatalogEntry } from "@/lib/features/catalog/types";
import type { Connection } from "@/lib/features/connections/types";
import { ACTION_LABEL, actionReasons, canOfferRemove, type RowAction } from "@/lib/features/deployments/actions";
import { compatibilityLine, nextExclusions, unsupportedBlocks, type BlockRef, type CompatibilityLine } from "@/lib/features/deployments/compatibility";
import { costLines, deployLine } from "@/lib/features/deployments/confirm";
import { useCompatibility, useRemoveDeployment, useStartRun, useUpdateDeployment } from "@/lib/features/deployments/hooks";
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
import { isActive, type DeploymentListItem, type DeploymentStatus, type RunAction, type RunStatus } from "@/lib/features/deployments/types";
import { CheckCircle2, ChevronDown, ChevronRight, FileCode2, KeyRound, Loader2, Play, Rocket, Trash2, TriangleAlert, X } from "lucide-react";
import { useId, useState } from "react";
import BlockPlacement from "./BlockPlacement";
import PreviewDialog from "./PreviewDialog";
import RunLog, { RunHeadline } from "./RunLog";
import SettingsForm from "./SettingsForm";
import WhatWasBuilt from "./WhatWasBuilt";

const DEPLOYMENT_STATUS: Record<DeploymentStatus, { label: string; className: string }> = {
  new: { label: "Not deployed", className: "border-slate-600 text-slate-300" },
  live: { label: "Deployed", className: "border-emerald-700 text-emerald-300" },
  failed: { label: "Needs attention", className: "border-red-700 text-red-300" },
  destroyed: { label: "Destroyed", className: "border-slate-600 text-slate-400" },
};

const ACTION: Record<RunAction, string> = { plan: "Plan", apply: "Deploy", destroy: "Destroy" };

const RUN_STATUS: Record<RunStatus, string> = { queued: "waiting to start", running: "running", completed: "finished", failed: "failed" };

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
};

const PART_TONE: Record<string, string> = {
  completed: "text-emerald-300",
  failed: "text-red-300",
  rolled_back: "text-amber-300",
  skipped: "text-slate-500",
};

export default function DeploymentRow({ token, organizationId, diagramId, item, entry, catalog, connections, blocks, blockTitle, webServers, blockedReason }: Props) {
  const cloud = entry?.manifest.title ?? item.connector;
  const settingsSpec = entry?.manifest.settings ?? [];
  const placements = item.placements ?? [];

  const compatQ = useCompatibility(token, organizationId, item.id);
  const updateM = useUpdateDeployment(token);
  // While a change to the exclusions is saving, show the change.
  const exclusions = (updateM.isPending ? updateM.variables?.data.exclusions : undefined) ?? item.exclusions ?? [];
  const startM = useStartRun(token);
  const removeM = useRemoveDeployment(token);
  const offId = useId();

  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [settings, setSettings] = useState<Record<string, unknown>>(() => ({ ...defaultValues(settingsSpec), ...(item.settings ?? {}) }));
  const [logOpen, setLogOpen] = useState(false);
  const [previewOpen, setPreviewOpen] = useState(false);
  const [confirm, setConfirm] = useState<"apply" | "destroy" | "remove" | null>(null);

  const run = item.latest_run;
  const running = isActive(run) || startM.isPending;
  const report = compatQ.data;
  const titleOf = (connector: string) => catalog.find((e) => e.manifest.connector === connector)?.manifest.title ?? connector;
  const manifestOf = (connector: string) => catalog.find((e) => e.manifest.connector === connector)?.manifest ?? null;
  const offers = (connector: string, kind: string) => {
    const m = manifestOf(connector);
    return !!m && manifestOffers(m, kind);
  };
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
          compatibilityLine(
            `${c.connection_name} (${titleOf(c.connector)})`,
            c,
            cloudBlocks(item.connection_id, placements, blocks, c.connection_id),
            exclusions,
            blockTitle,
          ),
        )
      : [compatibilityLine(cloud, report, blocks, exclusions, blockTitle)];
  const extra = report && clouds.length > 0 ? splitOnly(report) : { problems: [], notes: [] };
  const cross = report ? crossNotes(report) : { keys: [], addresses: [] };
  const byId = new Map(blocks.map((b) => [b.id, b.name]));
  const ready = !!report?.ready;
  const status = DEPLOYMENT_STATUS[item.status] ?? DEPLOYMENT_STATUS.new;

  // Blocks the own cloud cannot build, and the connection that can run each.
  const unsupported = entry ? unsupportedBlocks(entry.manifest, ownBlocks) : [];
  const moveReason = placementReason(item.status, running || updateM.isPending);
  const suggestions = new Map(placementSuggestions(unsupported, item.connection_id, connections, offers).map((s) => [s.block.id, s.connection]));

  // Settings are about some blocks (a function runtime about functions); only
  // those about this cloud's blocks show.
  const shownSettings = settingsFor(settingsSpec, new Set(ownBlocks.map((b) => b.kind)));
  const settingsErrors = valuesErrors(shownSettings, settings);

  const offersRemove = canOfferRemove(item.status);
  const offersDestroy = !offersRemove;

  const reasons = actionReasons({
    status: item.status,
    running,
    saving: updateM.isPending || removeM.isPending,
    unsaved: blockedReason,
    checking: compatQ.isLoading,
    checkFailed: compatQ.isError,
    ready,
  });

  const start = (action: RunAction) => {
    setConfirm(null);
    startM.mutate({ organizationId, diagramId, deploymentId: item.id, action });
  };

  const remove = () => {
    setConfirm(null);
    removeM.mutate({ organizationId, diagramId, deploymentId: item.id });
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
  const button = (action: RowAction) => ({ reason: reasons[action], help: help[action], describedBy: `${offId}-${action}` });

  return (
    <li data-testid="deployment-row" data-connector={item.connector} className="space-y-3 rounded-lg border border-slate-800 bg-slate-900/60 p-3">
      <div className="flex items-start justify-between gap-2">
        <div className="flex min-w-0 items-center gap-2.5">
          <CloudMark cloud={cloudLookFor(entry ? [entry] : [], item.connector)} size={24} />
          <div className="min-w-0">
            <div data-testid="deployment-title" className="break-words text-sm font-semibold text-white">
              {title}
            </div>
            <div data-testid="deployment-clouds" className="truncate text-xs text-slate-400" title={where}>
              {where}
            </div>
          </div>
        </div>
        <span className={`shrink-0 rounded border px-1.5 py-0.5 text-[11px] ${status.className}`}>{status.label}</span>
      </div>

      {/* Compatibility */}
      <div data-testid="compatibility">
        {compatQ.isLoading ? (
          <div role="status" className="flex items-center gap-2 text-xs text-slate-400">
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
            Checking what {cloud} can build…
          </div>
        ) : compatQ.isError ? (
          <p className="text-xs text-red-300">Could not check compatibility: {adapterErrorMessage(compatQ.error)}</p>
        ) : report ? (
          <div className="space-y-1.5">
            {lines.map((line) => (
              <div key={line.summary} className="space-y-1.5">
                <p className={`flex items-start gap-1.5 text-xs ${line.ready ? "text-emerald-300" : "text-amber-300"}`}>
                  {line.ready ? <CheckCircle2 className="mt-px h-3.5 w-3.5 shrink-0" /> : <TriangleAlert className="mt-px h-3.5 w-3.5 shrink-0" />}
                  {line.summary}
                </p>
                {line.choices.map((c) => {
                  const there = !c.supported && !moveReason ? suggestions.get(c.id) : undefined;
                  return (
                    <div key={c.id} className="space-y-1 pl-5">
                      {there && (
                        <button
                          type="button"
                          data-testid="run-on-suggestion"
                          disabled={updateM.isPending || running}
                          onClick={() => runOn(c.id, there.id)}
                          className="inline-flex items-center gap-1 rounded-md bg-teal-600 px-2 py-1 text-xs font-medium text-white hover:bg-teal-500 disabled:opacity-50">
                          Run {c.name} on {there.name}
                        </button>
                      )}
                      <label className="flex items-center gap-2 text-xs text-slate-200">
                        <input
                          type="checkbox"
                          className="h-4 w-4 accent-teal-500"
                          checked={c.excluded}
                          disabled={updateM.isPending || running}
                          onChange={(e) => toggleExclusion(c.id, e.target.checked)}
                        />
                        <span>
                          Leave {c.name} ({c.kindTitle}) out of this deployment
                          {!c.supported && !there && <span className="text-amber-300"> (not supported here)</span>}
                        </span>
                      </label>
                    </div>
                  );
                })}
                {line.problems.map((p) => (
                  <p key={p} className="pl-5 text-xs text-amber-200">
                    {p}
                  </p>
                ))}
                {line.notes.map((n) => (
                  <p key={n} className="pl-5 text-xs text-slate-400">
                    {n}
                  </p>
                ))}
              </div>
            ))}
            {extra.problems.map((p) => (
              <p key={p.code + p.message} className="flex items-start gap-1.5 text-xs text-amber-300">
                <TriangleAlert className="mt-px h-3.5 w-3.5 shrink-0" />
                {p.edge_source && p.edge_target ? `${byId.get(p.edge_source) ?? p.edge_source} to ${byId.get(p.edge_target) ?? p.edge_target}: ` : ""}
                {p.message}
              </p>
            ))}
            {extra.notes.map((n) => (
              <p key={n.code + n.message} className="pl-5 text-xs text-slate-400">
                {n.message}
              </p>
            ))}
            {cross.keys.map((k) => (
              <p key={k} data-testid="scoped-key-note" className="flex items-start gap-1.5 text-xs text-sky-200">
                <KeyRound className="mt-px h-3.5 w-3.5 shrink-0" aria-hidden />
                {k}
              </p>
            ))}
            {cross.addresses.map((a) => (
              <p key={a} className="pl-5 text-xs text-slate-400">
                {a}
              </p>
            ))}
          </div>
        ) : null}
        {updateM.isError && <p className="mt-1 text-xs text-red-300">{adapterErrorMessage(updateM.error)}</p>}
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
            className="flex items-center gap-1 text-xs text-slate-300 hover:text-white">
            {advancedOpen ? <ChevronDown className="h-3.5 w-3.5" aria-hidden /> : <ChevronRight className="h-3.5 w-3.5" aria-hidden />}
            Advanced
          </button>
          {advancedOpen && (
            <div className="mt-2 space-y-2 pl-1">
              <p className="text-[11px] text-slate-400">{cloud} settings</p>
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
              <button
                type="button"
                disabled={updateM.isPending || running || settingsErrors.length > 0}
                onClick={() =>
                  updateM.mutate({ organizationId, diagramId, deploymentId: item.id, data: { settings: cleanValues(settingsSpec, settings) } })
                }
                className="rounded-md bg-slate-700 px-2.5 py-1 text-xs text-white hover:bg-slate-600 disabled:opacity-50">
                {updateM.isPending ? "Saving…" : "Save settings"}
              </button>
            </div>
          )}
        </div>
      )}

      {/* Actions */}
      <div className="flex flex-wrap gap-1.5">
        <ActionButton label={ACTION_LABEL.preview} icon={<FileCode2 className="h-3.5 w-3.5" />} {...button("preview")} onClick={() => setPreviewOpen(true)} />
        <ActionButton label={ACTION_LABEL.plan} icon={<Play className="h-3.5 w-3.5" />} {...button("plan")} onClick={() => start("plan")} />
        <ActionButton label={ACTION_LABEL.apply} icon={<Rocket className="h-3.5 w-3.5" />} primary {...button("apply")} onClick={() => setConfirm("apply")} />
      </div>
      {confirm === "apply" && !reasons.apply && (
        <div role="dialog" aria-label="Deploy" data-testid="deploy-confirm" className="space-y-2 rounded-lg border border-teal-800/70 bg-teal-950/30 p-2.5">
          <p className="text-xs text-slate-100">{deployLine({ id: item.connection_id, name: item.connection_name }, placements, blocks, exclusions, blockTitle)}</p>
          {costLines(report).map((c) => (
            <p key={c} data-testid="cost-line" className="text-xs text-amber-200">
              {c}
            </p>
          ))}
          <div className="flex gap-1.5">
            <ActionButton label="Deploy" icon={<Rocket className="h-3.5 w-3.5" />} primary reason={null} help={help.apply} describedBy="" onClick={() => start("apply")} />
            <ActionButton label="Cancel" reason={null} help="" describedBy="" onClick={() => setConfirm(null)} />
          </div>
        </div>
      )}
      {/* Taking things away sits apart from building them. */}
      <div data-testid="remove-actions" className="flex flex-wrap gap-1.5 border-t border-slate-800 pt-2">
        {offersDestroy &&
          (confirm === "destroy" ? (
            <>
              <ActionButton label="Destroy for good" icon={<Trash2 className="h-3.5 w-3.5" />} confirmDanger {...button("destroy")} onClick={() => start("destroy")} />
              <ActionButton label="Keep it" reason={null} help="" describedBy="" onClick={() => setConfirm(null)} />
            </>
          ) : (
            <ActionButton label={ACTION_LABEL.destroy} icon={<Trash2 className="h-3.5 w-3.5" />} danger {...button("destroy")} onClick={() => setConfirm("destroy")} />
          ))}
        {confirm === "remove" ? (
          <>
            <ActionButton label="Remove for good" icon={<X className="h-3.5 w-3.5" />} confirmDanger {...button("remove")} onClick={remove} />
            <ActionButton label="Keep it" reason={null} help="" describedBy="" onClick={() => setConfirm(null)} />
          </>
        ) : (
          <ActionButton
            label={removeM.isPending ? "Removing…" : ACTION_LABEL.remove}
            icon={<X className="h-3.5 w-3.5" />}
            {...button("remove")}
            onClick={() => setConfirm("remove")}
          />
        )}
      </div>
      {confirm === "destroy" && <p className="text-xs text-red-300">Destroy removes everything this deployment built in {builtIn}. This cannot be undone.</p>}
      {/* Why each button is off, for screen readers; sighted users see it on hover. */}
      {(["preview", "plan", "apply", "destroy", "remove"] as const).map((a) =>
        reasons[a] ? (
          <span key={a} id={`${offId}-${a}`} className="sr-only">
            {ACTION_LABEL[a]} is off. {reasons[a]}
          </span>
        ) : null,
      )}
      {removeM.isError && (
        <p role="alert" className="text-xs text-red-300">
          {adapterErrorMessage(removeM.error)}
        </p>
      )}
      {startM.isError && (
        <div role="alert" className="text-xs text-red-300">
          <p>{adapterErrorMessage(startM.error)}</p>
          {startM.error instanceof AdapterApiError &&
            startM.error.problems.map((p, i) => (
              <p key={i} className="pl-3">
                {p.message}
              </p>
            ))}
        </div>
      )}

      {/* Latest run */}
      {run ? (
        <div className="space-y-1.5">
          <div className="flex items-center justify-between gap-2">
            <p data-testid="run-status" data-status={run.status} className="flex items-center gap-1.5 text-xs text-slate-300">
              {isActive(run) && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
              {ACTION[run.action]} {RUN_STATUS[run.status]}
              {run.finished_at && <span className="text-slate-500">at {new Date(run.finished_at).toLocaleTimeString()}</span>}
            </p>
            <button type="button" aria-expanded={logOpen} onClick={() => setLogOpen((o) => !o)} className="text-xs text-teal-300 hover:underline">
              {logOpen ? "Hide log" : "Show log"}
            </button>
          </div>
          {run.parts && run.parts.length > 0 && (
            <ol data-testid="run-parts" aria-label="Each cloud of this run" className="space-y-0.5 pl-5 text-xs">
              {partLines(run.parts).map((p) => (
                <li key={p.connectionId} className={`flex items-center gap-1.5 ${PART_TONE[p.status] ?? "text-slate-300"}`}>
                  {(p.status === "running" || p.status === "queued") && <Loader2 className="h-3 w-3 animate-spin" aria-hidden />}
                  {p.text}
                </li>
              ))}
            </ol>
          )}
          {!isActive(run) && <RunHeadline token={token} organizationId={organizationId} run={run} blocks={blocks.filter((b) => !(item.exclusions ?? []).includes(b.id)).length} />}
          {run.failure && <p className="text-xs text-red-300">{run.failure}</p>}
          {run.action === "apply" && run.status === "completed" && (
            <WhatWasBuilt
              token={token}
              organizationId={organizationId}
              item={item}
              run={run}
              catalog={catalog}
              blocks={blocks}
              blockTitle={blockTitle}
              webServers={webServers}
            />
          )}
          {logOpen && <RunLog token={token} organizationId={organizationId} run={run} />}
        </div>
      ) : (
        <p className="text-xs text-slate-500">No runs yet.</p>
      )}

      {previewOpen && (
        <PreviewDialog
          token={token}
          organizationId={organizationId}
          deploymentId={item.id}
          title={`${title} (${cloud})`}
          onClose={() => setPreviewOpen(false)}
        />
      )}
    </li>
  );
}

// A button that is off stays focusable (aria-disabled, not disabled), so a
// keyboard or screen-reader user reaches it and hears why (describedBy names
// a screen-reader line with the reason). Its tooltip says why it is off, or
// what it does.
function ActionButton({
  label,
  icon,
  reason,
  help,
  describedBy,
  onClick,
  primary,
  danger,
  confirmDanger,
}: {
  label: string;
  icon?: React.ReactNode;
  reason: string | null;
  help: string;
  describedBy: string;
  onClick: () => void;
  primary?: boolean;
  // Outlined red: a button that destroys. Solid red only on the confirm step.
  danger?: boolean;
  confirmDanger?: boolean;
}) {
  const off = !!reason;
  const tone = primary
    ? "bg-teal-600 text-white hover:bg-teal-500"
    : confirmDanger
      ? "bg-red-600 text-white hover:bg-red-500"
      : danger
        ? "border border-red-800 text-red-300 hover:bg-red-950"
        : "border border-slate-700 text-slate-200 hover:bg-slate-800";
  return (
    <button
      type="button"
      aria-disabled={off || undefined}
      aria-describedby={off && describedBy ? describedBy : undefined}
      title={reason ?? (help || undefined)}
      onClick={off ? undefined : onClick}
      className={`inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-teal-300 ${tone} aria-disabled:cursor-not-allowed aria-disabled:opacity-40`}>
      <span aria-hidden className="contents">
        {icon}
      </span>
      {label}
    </button>
  );
}
