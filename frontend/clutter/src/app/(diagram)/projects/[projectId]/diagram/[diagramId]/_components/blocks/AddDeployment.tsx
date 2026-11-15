"use client";

import CloudMark from "@/components/clouds/CloudMark";
import { adapterErrorMessage } from "@/lib/adapterApi";
import { cloudLookFor } from "@/lib/features/catalog/display";
import { cleanValues, defaultValues, valuesErrors } from "@/lib/features/catalog/forms";
import type { CatalogEntry } from "@/lib/features/catalog/types";
import type { Connection } from "@/lib/features/connections/types";
import { addDeploymentChoice } from "@/lib/features/deployments/actions";
import { unsupportedBlocks, type BlockRef } from "@/lib/features/deployments/compatibility";
import { useCreateDeployment } from "@/lib/features/deployments/hooks";
import { Loader2, Plus } from "lucide-react";
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
  blocks: BlockRef[];
  blockTitle: (kind: string) => string;
  disabledReason: string | null;
};

const linkClass = "text-teal-300 hover:underline";

// "Add deployment": bind this diagram to one of the organization's working
// connections, with that cloud's settings and the blocks it leaves out.
export default function AddDeployment({
  token,
  organizationId,
  diagramId,
  connections,
  usedConnectionIds,
  catalog,
  blocks,
  blockTitle,
  disabledReason,
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
          blocks={blocks}
          blockTitle={blockTitle}
          disabledReason={disabledReason}
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
  blocks,
  blockTitle,
  disabledReason,
  onDone,
}: {
  token: string | null;
  organizationId: string;
  diagramId: string;
  connection: Connection;
  entry: CatalogEntry;
  blocks: BlockRef[];
  blockTitle: (kind: string) => string;
  disabledReason: string | null;
  onDone: () => void;
}) {
  const settingsSpec = entry.manifest.settings ?? [];
  const unsupported = unsupportedBlocks(entry.manifest, blocks);
  const [settings, setSettings] = useState<Record<string, unknown>>(() => defaultValues(settingsSpec));
  const [leaveOut, setLeaveOut] = useState<string[]>([]);
  const createM = useCreateDeployment(token);
  const errors = valuesErrors(settingsSpec, settings);

  const submit = () =>
    createM.mutate(
      {
        organizationId,
        data: { diagram_id: diagramId, connection_id: connection.id, settings: cleanValues(settingsSpec, settings), exclusions: leaveOut },
      },
      { onSuccess: onDone },
    );

  return (
    <div className="space-y-3">
      {unsupported.length === 0 ? (
        <p className="text-xs text-emerald-300">{entry.manifest.title} can build every block in this diagram.</p>
      ) : (
        <div className="space-y-1.5">
          <p className="text-xs text-amber-300">
            {entry.manifest.title} cannot build {unsupported.length === 1 ? "1 block" : `${unsupported.length} blocks`}. Leave them out of
            this deployment, or remove them from the diagram.
          </p>
          {unsupported.map((b) => (
            <label key={b.id} className="flex items-center gap-2 text-xs text-slate-200">
              <input
                type="checkbox"
                className="h-4 w-4 accent-teal-500"
                checked={leaveOut.includes(b.id)}
                onChange={(e) => setLeaveOut((cur) => (e.target.checked ? [...cur, b.id] : cur.filter((x) => x !== b.id)))}
              />
              Leave out {b.name} ({blockTitle(b.kind)})
            </label>
          ))}
        </div>
      )}
      {settingsSpec.length > 0 && (
        <div className="space-y-2">
          <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500">{entry.manifest.title} settings</h4>
          {settingsSpec.map((f) => (
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
      {createM.isError && (
        <p role="alert" className="text-sm text-red-300">
          {adapterErrorMessage(createM.error)}
        </p>
      )}
      {disabledReason && <p className="text-xs text-amber-300">{disabledReason}</p>}
      <button
        type="button"
        disabled={createM.isPending || errors.length > 0 || !!disabledReason}
        onClick={submit}
        className="inline-flex items-center gap-1.5 rounded-md bg-teal-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-teal-500 disabled:opacity-50">
        {createM.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
        Add deployment to {connection.name}
      </button>
    </div>
  );
}
