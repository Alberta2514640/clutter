"use client";

import CloudLogo from "@/components/common/CloudLogo";
import { Button } from "@/components/ui/button";
import { toast } from "@/components/ui/sonner";
import { cn } from "@/lib/utils";
import { adapterErrorMessage } from "@/lib/adapterApi";
import { cloudLookFor } from "@/lib/features/catalog/display";
import { cleanValues, cloudOffers, defaultValues, entryFor, settingsFor, valuesErrors } from "@/lib/features/catalog/forms";
import type { CatalogEntry } from "@/lib/features/catalog/types";
import { useConnectionOptions } from "@/lib/features/connections/hooks";
import { applySuggestions } from "@/lib/features/connections/options";
import type { Connection, ConnectionOptions } from "@/lib/features/connections/types";
import { addDeploymentChoice, startingConnection } from "@/lib/features/deployments/actions";
import { unsupportedBlocks, type BlockRef } from "@/lib/features/deployments/compatibility";
import { useCreateDeployment } from "@/lib/features/deployments/hooks";
import {
  examplePresets,
  exampleSplitFor,
  missingExampleClouds,
  placementSuggestions,
  suggestedPlacements,
  type ExampleSplit,
} from "@/lib/features/deployments/placements";
import { useExamples } from "@/lib/features/examples/hooks";
import { ChevronDown, ChevronRight, Loader2, Plus } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import ConnectionsLink from "./ConnectionsLink";
import SettingsForm from "./SettingsForm";

type Props = {
  token: string | null;
  organizationId: string;
  diagramId: string;
  // All of the organization's connections; those in usedConnectionIds already
  // have a deployment of this diagram.
  connections: Connection[];
  usedConnectionIds: ReadonlySet<string>;
  catalog: CatalogEntry[];
  // The blocks on the canvas, saved or not: the deployment is for them.
  blocks: BlockRef[];
  blockTitle: (kind: string) => string;
  // Saves the canvas before the deployment is added, when it has changes;
  // false when the save failed or cannot run (the editor says why).
  saveFirst: (() => Promise<boolean>) | null;
  // Opens the connection flow in a side sheet, without leaving the editor.
  onAddConnection: () => void;
  // Called with the new deployment's id once it exists.
  onAdded: (deploymentId: string) => void;
};

// py-0.5 and leading-5 make each link in a sentence a 24px target.
const linkClass = "text-link inline-block py-0.5 font-medium leading-5 hover:underline";

// "Add deployment": bind this diagram to one of the organization's working
// connections, with that cloud's settings, and for each block it cannot
// build, another connection that runs it or leaving it out.
export default function AddDeployment({
  token,
  organizationId,
  diagramId,
  connections,
  usedConnectionIds,
  catalog,
  blocks,
  blockTitle,
  saveFirst,
  onAddConnection,
  onAdded,
}: Props) {
  const [open, setOpen] = useState(false);
  const [connectionId, setConnectionId] = useState("");
  // The form takes focus as it opens (its button is gone), so Escape reaches
  // it; Escape and Cancel close it and put focus back on "Add deployment".
  const addRef = useRef<HTMLButtonElement>(null);
  const formRef = useRef<HTMLDivElement>(null);
  const refocus = useRef(false);
  useEffect(() => {
    if (open) formRef.current?.focus();
    else if (refocus.current) {
      refocus.current = false;
      addRef.current?.focus();
    }
  }, [open]);
  const close = () => {
    refocus.current = true;
    setOpen(false);
  };
  const choice = addDeploymentChoice(connections, usedConnectionIds);
  // A diagram made from a split example (Hybrid) starts on the example's
  // clouds. Without the examples, the form works as for any diagram.
  const examplesQ = useExamples(token);
  const split = exampleSplitFor(examplesQ.data ?? [], blocks);
  const shortName = (connector: string) => cloudLookFor(catalog, connector).shortName;
  const splitLine = split
    ? `This example runs on two clouds: ${[...split.placed]
        .map(([id, cloud]) => `${blocks.find((b) => b.id === id)?.name ?? id} on ${shortName(cloud)}`)
        .join(", ")}, the rest on ${shortName(split.primary)}.`
    : null;

  // A free connection that failed its last check, named so the user knows
  // why it is not offered.
  const brokenLines = choice.broken.map((c) => (
    <p key={c.id} className="text-warning text-xs">
      {c.name} is not working: verify it on{" "}
      <ConnectionsLink organizationId={organizationId} className={linkClass}>
        Connections
      </ConnectionsLink>
      .
    </p>
  ));
  const addConnection = (
    <button type="button" onClick={onAddConnection} className={linkClass}>
      Add a connection
    </button>
  );

  if (choice.state !== "pick") {
    return (
      <div data-testid="add-deployment-none" className="space-y-1.5 rounded-lg border border-dashed px-3 py-2.5">
        {choice.state === "none_working" ? (
          <>
            {brokenLines}
            <p className="text-muted-foreground text-xs">Or {addConnection} for another cloud.</p>
          </>
        ) : (
          <p className="text-muted-foreground text-sm">
            {choice.state === "none" ? "No connections yet." : "All your connections already have a deployment here."} {addConnection} to deploy to another
            cloud.
          </p>
        )}
      </div>
    );
  }

  if (!open) {
    return (
      <div className="space-y-2">
        <Button
          ref={addRef}
          type="button"
          variant="outline"
          className="w-full border-dashed"
          onClick={() => {
            setOpen(true);
            // Start on a connection of the example's first cloud, else on the
            // only free connection when there is one.
            const first = startingConnection(choice.usable, split?.primary ?? null);
            if (first) setConnectionId(first);
          }}>
          <Plus aria-hidden />
          Add deployment
        </Button>
        <p className="text-muted-foreground text-center text-xs">Need another cloud? {addConnection}.</p>
      </div>
    );
  }

  const usable = choice.usable;
  const connection = usable.find((c) => c.id === connectionId) ?? null;
  const entry = entryFor(catalog, connection?.connector);

  return (
    <div
      ref={formRef}
      tabIndex={-1}
      role="group"
      aria-label="Add deployment"
      className="surface-card space-y-4 p-4 outline-none animate-fade-in"
      onKeyDown={(e) => {
        // Only an Escape from inside the form itself: a list or menu opened
        // from it (a settings Select, in a portal) closes on its own Escape.
        if (e.key === "Escape" && !e.defaultPrevented && e.currentTarget.contains(e.target as Node)) close();
      }}>
      <div>
        <h3 className="text-sm font-semibold">Add deployment</h3>
        <p className="text-muted-foreground text-xs">Choose the connection this diagram builds on.</p>
      </div>
      {splitLine && (
        <p data-testid="example-split" className="text-info text-xs">
          {splitLine}
        </p>
      )}
      {brokenLines}
      <fieldset className="space-y-2">
        <legend className="mb-1.5 text-xs font-medium">Choose a connection</legend>
        {usable.map((c) => {
          const cloud = cloudLookFor(catalog, c.connector);
          const checked = c.id === connectionId;
          return (
            <label
              key={c.id}
              className={cn(
                "has-[:focus-visible]:ring-ring/50 flex cursor-pointer items-center gap-3 rounded-lg border px-3 py-2.5 transition-colors has-[:focus-visible]:ring-[3px]",
                checked ? "border-primary bg-primary/10" : "hover:bg-accent/50",
              )}>
              <input
                type="radio"
                name={`add-deployment-${diagramId}`}
                value={c.id}
                checked={checked}
                onChange={() => setConnectionId(c.id)}
                className="sr-only"
              />
              <CloudLogo connector={cloud.connector} shortName={cloud.shortName} src={cloud.logoSrc} size={22} />
              <span className="min-w-0">
                <span className="block truncate text-sm font-medium">{c.name}</span>
                <span className="text-muted-foreground block text-xs">
                  {cloud.title}, {c.region}
                </span>
              </span>
            </label>
          );
        })}
      </fieldset>
      {connection && !entry && <p className="text-danger text-sm">Clutter no longer offers this connection&apos;s cloud.</p>}
      {connection && entry && (
        <DeploymentForm
          key={connection.id}
          token={token}
          organizationId={organizationId}
          diagramId={diagramId}
          connection={connection}
          entry={entry}
          catalog={catalog}
          connections={connections}
          blocks={blocks}
          blockTitle={blockTitle}
          split={split}
          saveFirst={saveFirst}
          onDone={(id) => {
            setOpen(false);
            setConnectionId("");
            onAdded(id);
          }}
          onCancel={close}
        />
      )}
      {!(connection && entry) && (
        <Button type="button" variant="ghost" size="sm" onClick={close}>
          Cancel
        </Button>
      )}
    </div>
  );
}

function DeploymentForm({
  token,
  organizationId,
  diagramId,
  connection,
  entry,
  catalog,
  connections,
  blocks,
  blockTitle,
  split,
  saveFirst,
  onDone,
  onCancel,
}: {
  token: string | null;
  organizationId: string;
  diagramId: string;
  connection: Connection;
  entry: CatalogEntry;
  catalog: CatalogEntry[];
  connections: Connection[];
  blocks: BlockRef[];
  blockTitle: (kind: string) => string;
  split: ExampleSplit | null;
  saveFirst: (() => Promise<boolean>) | null;
  onDone: (deploymentId: string) => void;
  onCancel: () => void;
}) {
  const settingsSpec = entry.manifest.settings ?? [];
  const unsupported = unsupportedBlocks(entry.manifest, blocks);
  const offers = (connector: string, kind: string) => cloudOffers(catalog, connector, kind);
  // Where each block goes unless the person says otherwise: the example's
  // own cloud for it, else the first other connection that can build a block
  // this cloud cannot.
  const presets = examplePresets(split, connection, connections, offers, blocks);
  const suggested = new Map([
    ...placementSuggestions(unsupported, connection.id, connections, offers).map((s) => [s.block.id, s.connection] as const),
    ...presets,
  ]);
  // The blocks to choose for: those this cloud cannot build, and those the
  // example puts on another cloud.
  const choosing = blocks.filter((b) => unsupported.includes(b) || presets.has(b.id));
  // A split example whose other cloud the team has no working connection to
  // cannot split here; say so rather than place every block here silently.
  const missing = missingExampleClouds(split, connections).filter((c) => c !== connection.connector);
  const missingLine =
    missing.length === 0
      ? null
      : `No ${missing.map((c) => cloudLookFor(catalog, c).shortName).join(" or ")} connection: ${
          choosing.length === 0 ? `every block runs on ${cloudLookFor(catalog, connection.connector).shortName} here.` : "this deployment cannot split as the example does."
        }`;
  // Settings about blocks the new deployment's own cloud runs, in Advanced.
  const ownKinds = new Set(blocks.filter((b) => !unsupported.includes(b)).map((b) => b.kind));
  const shownSettings = settingsFor(settingsSpec, ownKinds);
  const [settings, setSettings] = useState<Record<string, unknown>>(() => defaultValues(settingsSpec));
  // The connection's own lists, read as the form opens. A setting still at a
  // default this cloud lacks (DevStack's cirros on a public cloud) takes the
  // cloud's suggestion, so the first plan works without opening Advanced.
  // Each answer is applied once; a value the user chose is never replaced.
  const wantsLists = settingsSpec.some((f) => f.type === "string");
  const optionsQ = useConnectionOptions(token, organizationId, wantsLists ? connection.id : null);
  const [appliedFrom, setAppliedFrom] = useState<ConnectionOptions | null>(null);
  const [picked, setPicked] = useState<Record<string, unknown>>({});
  if (optionsQ.data && optionsQ.data !== appliedFrom) {
    setAppliedFrom(optionsQ.data);
    const s = applySuggestions(settingsSpec, settings, optionsQ.data);
    if (s.picked.length > 0) {
      setSettings(s.values);
      setPicked((p) => ({ ...p, ...Object.fromEntries(s.picked.map((name) => [name, s.values[name]])) }));
    }
  }
  // Shown settings that still hold the value picked from the cloud's list.
  const pickedShown = shownSettings.filter((f) => f.name in picked && settings[f.name] === picked[f.name]);
  const listing = wantsLists && optionsQ.isLoading;
  // For each block this cloud cannot build: "run" on the suggested
  // connection (the default when there is one), or "out".
  const [choice, setChoice] = useState<Record<string, "run" | "out" | "keep">>({});
  const choiceOf = (id: string) => choice[id] ?? (suggested.has(id) ? "run" : "keep");
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const createM = useCreateDeployment(token);
  const errors = valuesErrors(shownSettings, settings);

  const submit = async () => {
    if (saveFirst) {
      setSaving(true);
      const ok = await saveFirst();
      setSaving(false);
      if (!ok) return;
    }
    const run = choosing.filter((b) => choiceOf(b.id) === "run" && suggested.has(b.id));
    createM.mutate(
      {
        organizationId,
        data: {
          diagram_id: diagramId,
          connection_id: connection.id,
          settings: cleanValues(settingsSpec, settings),
          exclusions: unsupported.filter((b) => choiceOf(b.id) === "out").map((b) => b.id),
          placements: suggestedPlacements(run.map((b) => ({ blockId: b.id, connectionId: suggested.get(b.id)!.id }))),
        },
      },
      {
        onSuccess: (created) => {
          toast.success(`Deployment added on ${connection.name}`);
          onDone(created.id);
        },
      },
    );
  };

  const n = unsupported.length;
  const busy = saving || createM.isPending;

  return (
    <div className="space-y-3">
      {missingLine && (
        <p data-testid="example-cloud-missing" className="text-warning text-xs">
          {missingLine}
        </p>
      )}
      {choosing.length === 0 ? (
        <p className="text-success text-xs">{entry.manifest.title} can build every block in this diagram.</p>
      ) : (
        <div className="space-y-2">
          {n > 0 && (
            <p className="text-warning text-xs">
              {entry.manifest.title} cannot build {n === 1 ? "1 block" : `${n} blocks`}.{" "}
              {suggested.size > 0
                ? `Choose where ${n === 1 ? "it runs" : "they run"}.`
                : `Leave ${n === 1 ? "it" : "them"} out of this deployment, or remove ${n === 1 ? "it" : "them"} from the diagram.`}
            </p>
          )}
          {choosing.map((b) => {
            const there = suggested.get(b.id);
            if (!there) {
              return (
                <label key={b.id} className="flex min-h-6 items-center gap-2 text-xs">
                  <input
                    type="checkbox"
                    className="accent-primary size-4"
                    checked={choiceOf(b.id) === "out"}
                    onChange={(e) => setChoice((c) => ({ ...c, [b.id]: e.target.checked ? "out" : "keep" }))}
                  />
                  Leave out {b.name} ({blockTitle(b.kind)})
                </label>
              );
            }
            // A block this cloud can build stays here as the other choice.
            const other = unsupported.includes(b) ? ("out" as const) : ("keep" as const);
            return (
              <fieldset key={b.id} className="space-y-1">
                <legend className="sr-only">
                  Where {b.name} ({blockTitle(b.kind)}) runs
                </legend>
                {(["run", other] as const).map((v) => (
                  <label key={v} className="flex min-h-6 items-center gap-2 text-xs">
                    <input
                      type="radio"
                      name={`placement-${connection.id}-${b.id}`}
                      className="accent-primary size-4"
                      checked={choiceOf(b.id) === v}
                      onChange={() => setChoice((c) => ({ ...c, [b.id]: v }))}
                    />
                    {v === "run" ? `Run ${b.name} on ${there.name}` : v === "out" ? `Leave ${b.name} out` : `Run ${b.name} on ${connection.name}`}
                  </label>
                ))}
              </fieldset>
            );
          })}
        </div>
      )}
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
            <div className="mt-2 space-y-3">
              {pickedShown.length > 0 && (
                <p data-testid="settings-from-cloud" className="text-info text-[11px]">
                  {pickedShown.length === 1 ? "One value below comes" : "Some values below come"} from your cloud&apos;s own lists,
                  because it lacks the usual defaults. You can change them.
                </p>
              )}
              <SettingsForm
                token={token}
                organizationId={organizationId}
                connectionId={connection.id}
                idPrefix={`new-deployment-${connection.id}`}
                spec={shownSettings}
                values={settings}
                onChange={(name, v) => setSettings((s) => ({ ...s, [name]: v }))}
              />
            </div>
          )}
        </div>
      )}
      {createM.isError && (
        <p role="alert" className="text-danger text-sm">
          {adapterErrorMessage(createM.error)}
        </p>
      )}
      {listing && (
        <p role="status" className="text-muted-foreground flex items-center gap-1.5 text-[11px]">
          <Loader2 className="size-3 animate-spin" aria-hidden />
          Reading the choices your cloud offers…
        </p>
      )}
      <div className="flex flex-wrap items-center gap-2">
        <Button type="button" size="sm" disabled={busy || listing || errors.length > 0} onClick={() => void submit()}>
          {busy && <Loader2 className="animate-spin" aria-hidden />}
          {saving ? "Saving the diagram…" : `Add deployment to ${connection.name}`}
        </Button>
        <Button type="button" variant="ghost" size="sm" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </div>
  );
}
