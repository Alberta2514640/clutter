"use client";

import CloudMark from "@/components/clouds/CloudMark";
import { adapterErrorMessage } from "@/lib/adapterApi";
import { cloudLookFor } from "@/lib/features/catalog/display";
import { cleanValues, defaultValues, manifestOffers, settingsFor, valuesErrors } from "@/lib/features/catalog/forms";
import type { CatalogEntry } from "@/lib/features/catalog/types";
import type { Connection } from "@/lib/features/connections/types";
import { addDeploymentChoice } from "@/lib/features/deployments/actions";
import { unsupportedBlocks, type BlockRef } from "@/lib/features/deployments/compatibility";
import { useCreateDeployment } from "@/lib/features/deployments/hooks";
import { placementSuggestions, suggestedPlacements } from "@/lib/features/deployments/placements";
import { ChevronDown, ChevronRight, Loader2, Plus } from "lucide-react";
import { useState } from "react";
import ConnectionsLink from "./ConnectionsLink";
import FieldInput from "./FieldInput";

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
};

const linkClass = "text-teal-300 hover:underline";

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
}: Props) {
  const [open, setOpen] = useState(false);
  const [connectionId, setConnectionId] = useState("");
  const choice = addDeploymentChoice(connections, usedConnectionIds);

  // A free connection that failed its last check, named so the user knows
  // why it is not offered.
  const brokenLines = choice.broken.map((c) => (
    <p key={c.id} className="text-xs text-amber-300">
      {c.name} is not working: verify it on{" "}
      <ConnectionsLink organizationId={organizationId} className={linkClass}>
        Connections
      </ConnectionsLink>
      .
    </p>
  ));

  if (choice.state !== "pick") {
    return (
      <div data-testid="add-deployment-none" className="space-y-1.5 rounded-lg border border-dashed border-slate-700 px-3 py-2.5">
        {choice.state === "none_working" ? (
          brokenLines
        ) : (
          <p className="text-sm text-slate-300">
            {choice.state === "none" ? "No connections yet." : "All your connections already have a deployment here."}{" "}
            <ConnectionsLink organizationId={organizationId} className={linkClass}>
              Add a connection
            </ConnectionsLink>{" "}
            to deploy to another cloud.
          </p>
        )}
      </div>
    );
  }

  if (!open) {
    return (
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="inline-flex w-full items-center justify-center gap-1.5 rounded-lg border border-dashed border-slate-600 px-3 py-2 text-sm text-slate-200 hover:border-teal-500 hover:text-white">
        <Plus className="h-4 w-4" aria-hidden />
        Add deployment
      </button>
    );
  }

  const usable = choice.usable;
  const connection = usable.find((c) => c.id === connectionId) ?? null;
  const entry = connection ? (catalog.find((e) => e.manifest.connector === connection.connector) ?? null) : null;

  return (
    <div className="space-y-3 rounded-lg border border-slate-700 bg-slate-900/60 p-3">
      <h3 className="text-sm font-semibold text-white">Add deployment</h3>
      {brokenLines}
      <fieldset className="space-y-1.5">
        <legend className="mb-1 text-xs font-medium text-slate-300">Choose a connection</legend>
        {usable.map((c) => {
          const cloud = cloudLookFor(catalog, c.connector);
          const checked = c.id === connectionId;
          return (
            <label
              key={c.id}
              className={`flex cursor-pointer items-center gap-2.5 rounded-lg border px-2.5 py-2 has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-teal-400 ${
                checked ? "border-teal-500 bg-teal-500/10" : "border-slate-700 bg-slate-900/60 hover:border-slate-500"
              }`}>
              <input
                type="radio"
                name={`add-deployment-${diagramId}`}
                value={c.id}
                checked={checked}
                onChange={() => setConnectionId(c.id)}
                className="sr-only"
              />
              <CloudMark cloud={cloud} size={20} />
              <span className="min-w-0">
                <span className="block truncate text-sm text-white">{c.name}</span>
                <span className="block text-xs text-slate-400">
                  {cloud.title}, {c.region}
                </span>
              </span>
            </label>
          );
        })}
      </fieldset>
      {connection && !entry && <p className="text-sm text-red-300">Clutter no longer offers this connection&apos;s cloud.</p>}
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
          saveFirst={saveFirst}
          onDone={() => {
            setOpen(false);
            setConnectionId("");
          }}
        />
      )}
      <button type="button" onClick={() => setOpen(false)} className="text-xs text-slate-400 hover:text-white">
        Cancel
      </button>
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
  saveFirst,
  onDone,
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
  saveFirst: (() => Promise<boolean>) | null;
  onDone: () => void;
}) {
  const settingsSpec = entry.manifest.settings ?? [];
  const unsupported = unsupportedBlocks(entry.manifest, blocks);
  const offers = (connector: string, kind: string) => {
    const m = catalog.find((e) => e.manifest.connector === connector)?.manifest;
    return !!m && manifestOffers(m, kind);
  };
  const suggested = new Map(placementSuggestions(unsupported, connection.id, connections, offers).map((s) => [s.block.id, s.connection]));
  // Settings about blocks the new deployment's own cloud runs, in Advanced.
  const ownKinds = new Set(blocks.filter((b) => !unsupported.includes(b)).map((b) => b.kind));
  const shownSettings = settingsFor(settingsSpec, ownKinds);
  const [settings, setSettings] = useState<Record<string, unknown>>(() => defaultValues(settingsSpec));
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
    const run = unsupported.filter((b) => choiceOf(b.id) === "run" && suggested.has(b.id));
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
      { onSuccess: onDone },
    );
  };

  const n = unsupported.length;
  const busy = saving || createM.isPending;

  return (
    <div className="space-y-3">
      {n === 0 ? (
        <p className="text-xs text-emerald-300">{entry.manifest.title} can build every block in this diagram.</p>
      ) : (
        <div className="space-y-2">
          <p className="text-xs text-amber-300">
            {entry.manifest.title} cannot build {n === 1 ? "1 block" : `${n} blocks`}.{" "}
            {suggested.size > 0
              ? `Choose where ${n === 1 ? "it runs" : "they run"}.`
              : `Leave ${n === 1 ? "it" : "them"} out of this deployment, or remove ${n === 1 ? "it" : "them"} from the diagram.`}
          </p>
          {unsupported.map((b) => {
            const there = suggested.get(b.id);
            if (!there) {
              return (
                <label key={b.id} className="flex items-center gap-2 text-xs text-slate-200">
                  <input
                    type="checkbox"
                    className="h-4 w-4 accent-teal-500"
                    checked={choiceOf(b.id) === "out"}
                    onChange={(e) => setChoice((c) => ({ ...c, [b.id]: e.target.checked ? "out" : "keep" }))}
                  />
                  Leave out {b.name} ({blockTitle(b.kind)})
                </label>
              );
            }
            return (
              <fieldset key={b.id} className="space-y-1">
                <legend className="sr-only">
                  Where {b.name} ({blockTitle(b.kind)}) runs
                </legend>
                {(["run", "out"] as const).map((v) => (
                  <label key={v} className="flex items-center gap-2 text-xs text-slate-200">
                    <input
                      type="radio"
                      name={`placement-${connection.id}-${b.id}`}
                      className="h-4 w-4 accent-teal-500"
                      checked={choiceOf(b.id) === v}
                      onChange={() => setChoice((c) => ({ ...c, [b.id]: v }))}
                    />
                    {v === "run" ? `Run ${b.name} on ${there.name}` : `Leave ${b.name} out`}
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
            className="flex items-center gap-1 text-xs text-slate-300 hover:text-white">
            {advancedOpen ? <ChevronDown className="h-3.5 w-3.5" aria-hidden /> : <ChevronRight className="h-3.5 w-3.5" aria-hidden />}
            Advanced
          </button>
          {advancedOpen && (
            <div className="mt-2 space-y-2">
              {shownSettings.map((f) => (
                <FieldInput
                  key={f.name}
                  id={`new-deployment-${connection.id}-${f.name}`}
                  field={f}
                  value={settings[f.name]}
                  onChange={(v) => setSettings((s) => ({ ...s, [f.name]: v }))}
                />
              ))}
            </div>
          )}
        </div>
      )}
      {createM.isError && (
        <p role="alert" className="text-sm text-red-300">
          {adapterErrorMessage(createM.error)}
        </p>
      )}
      <button
        type="button"
        disabled={busy || errors.length > 0}
        onClick={() => void submit()}
        className="inline-flex items-center gap-1.5 rounded-md bg-teal-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-teal-500 disabled:opacity-50">
        {busy && <Loader2 className="h-4 w-4 animate-spin" />}
        {saving ? "Saving the diagram…" : `Add deployment to ${connection.name}`}
      </button>
    </div>
  );
}
