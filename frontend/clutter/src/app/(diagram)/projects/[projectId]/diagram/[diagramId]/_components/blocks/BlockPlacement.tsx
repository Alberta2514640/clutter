"use client";

import { adapterErrorMessage } from "@/lib/adapterApi";
import { cleanValues, defaultValues, valuesErrors } from "@/lib/features/catalog/forms";
import type { CatalogEntry } from "@/lib/features/catalog/types";
import type { Connection } from "@/lib/features/connections/types";
import type { BlockRef } from "@/lib/features/deployments/compatibility";
import { useUpdateDeployment } from "@/lib/features/deployments/hooks";
import { blockCloud, cloudChoices, movePlacement, placementReason, withPlacementSettings } from "@/lib/features/deployments/placements";
import type { DeploymentListItem, Placement } from "@/lib/features/deployments/types";
import { ChevronDown, ChevronRight } from "lucide-react";
import { useState } from "react";
import SettingsForm from "./SettingsForm";

type Props = {
  token: string | null;
  organizationId: string;
  diagramId: string;
  item: DeploymentListItem;
  catalog: CatalogEntry[];
  connections: Connection[];
  blocks: BlockRef[];
  blockTitle: (kind: string) => string;
  running: boolean;
};

// BlockPlacement is a deployment's "Cloud" choice per block (ADR-020): each
// block runs on the deployment's own connection unless it is placed on
// another of the organization's connections, which has its own settings. B
// checks that every connection is the organization's and that each edge can
// cross clouds; the compatibility line above says what it found.
export default function BlockPlacement({ token, organizationId, diagramId, item, catalog, connections, blocks, blockTitle, running }: Props) {
  const updateM = useUpdateDeployment(token);
  const [open, setOpen] = useState(false);
  const placements: Placement[] = item.placements ?? [];
  const entryOf = (connector: string) => catalog.find((e) => e.manifest.connector === connector) ?? null;
  const primary = connections.find((c) => c.id === item.connection_id);
  const choices = cloudChoices(primary, connections, (c) => !!entryOf(c), new Set(placements.map((p) => p.connection_id)));
  const reason = placementReason(item.status, running || updateM.isPending);

  // With one usable cloud there is nothing to choose.
  if (choices.length < 2 && placements.length === 0) return null;

  const label = (c: Connection) => `${c.name} (${entryOf(c.connector)?.manifest.display.short_name || c.connector})`;
  const move = (blockId: string, connectionId: string) =>
    updateM.mutate({
      organizationId,
      diagramId,
      deploymentId: item.id,
      data: { placements: movePlacement(item.connection_id, placements, blocks, blockId, connectionId) },
    });

  return (
    <div>
      <button
        type="button"
        aria-expanded={open}
        onClick={() => setOpen((o) => !o)}
        className="flex items-center gap-1 text-xs text-slate-300 hover:text-white">
        {open ? <ChevronDown className="h-3.5 w-3.5" aria-hidden /> : <ChevronRight className="h-3.5 w-3.5" aria-hidden />}
        Where each block runs
        {placements.length > 0 && (
          <span className="text-slate-500">
            {" "}
            ({placements.length + 1} clouds)
          </span>
        )}
      </button>
      {open && (
        <div className="mt-2 space-y-2 pl-1" data-testid="block-placement">
          <p className="text-[11px] leading-snug text-slate-400">
            Every block runs on {item.connection_name} unless you choose another cloud for it. Clutter builds the clouds in the order the
            links between their blocks need.
          </p>
          {reason && <p className="text-xs text-amber-300">{reason}</p>}
          {blocks.length === 0 ? (
            <p className="text-xs text-slate-500">Save the diagram with blocks first.</p>
          ) : (
            // One row per block: its name and kind on a line, its cloud below at the
            // panel's full width, so neither the name nor the cloud is clipped.
            <ul className="space-y-2 text-xs" aria-label="The cloud each block runs on">
              {blocks.map((b) => {
                const selectId = `${item.id}-cloud-${b.id}`;
                const current = blockCloud(item.connection_id, placements, b.id);
                const currentChoice = choices.find((c) => c.id === current);
                return (
                  <li key={b.id} className="space-y-1">
                    <label htmlFor={selectId} className="flex min-w-0 items-baseline gap-1 text-slate-200" title={`${b.name} (${blockTitle(b.kind)})`}>
                      <span className="truncate">{b.name}</span>
                      <span className="shrink-0 whitespace-nowrap text-slate-500">({blockTitle(b.kind)})</span>
                    </label>
                    <select
                      id={selectId}
                      value={current}
                      disabled={!!reason}
                      title={currentChoice ? label(currentChoice) : undefined}
                      aria-describedby={reason ? `${item.id}-placement-reason` : undefined}
                      onChange={(e) => move(b.id, e.target.value)}
                      className="w-full truncate rounded border border-slate-700 bg-slate-950 px-1.5 py-1 text-xs text-slate-100 disabled:opacity-50">
                      {choices.map((c) => (
                        <option key={c.id} value={c.id}>
                          {label(c)}
                          {c.id === item.connection_id ? " (this deployment's own)" : ""}
                        </option>
                      ))}
                    </select>
                  </li>
                );
              })}
            </ul>
          )}
          {reason && (
            <span id={`${item.id}-placement-reason`} className="sr-only">
              {reason}
            </span>
          )}
          {updateM.isPending && <p className="text-xs text-slate-400">Saving…</p>}
          {updateM.isError && (
            <p role="alert" className="text-xs text-red-300">
              {adapterErrorMessage(updateM.error)}
            </p>
          )}
          {placements.map((p) => (
            <PlacedCloudSettings
              key={p.connection_id}
              token={token}
              organizationId={organizationId}
              diagramId={diagramId}
              item={item}
              placement={p}
              entry={entryOf(p.connector)}
              disabled={running}
            />
          ))}
        </div>
      )}
    </div>
  );
}

// PlacedCloudSettings edits one placed cloud's own settings (its image, its
// instance type). They may change while the deployment is live.
function PlacedCloudSettings({
  token,
  organizationId,
  diagramId,
  item,
  placement,
  entry,
  disabled,
}: {
  token: string | null;
  organizationId: string;
  diagramId: string;
  item: DeploymentListItem;
  placement: Placement;
  entry: CatalogEntry | null;
  disabled: boolean;
}) {
  const spec = entry?.manifest.settings ?? [];
  const updateM = useUpdateDeployment(token);
  const [open, setOpen] = useState(false);
  const [values, setValues] = useState<Record<string, unknown>>(() => ({ ...defaultValues(spec), ...(placement.settings ?? {}) }));
  if (spec.length === 0) return null;
  const errors = valuesErrors(spec, values);
  return (
    <div>
      <button type="button" aria-expanded={open} onClick={() => setOpen((o) => !o)} className="flex items-center gap-1 text-xs text-slate-300 hover:text-white">
        {open ? <ChevronDown className="h-3.5 w-3.5" aria-hidden /> : <ChevronRight className="h-3.5 w-3.5" aria-hidden />}
        {placement.connection_name} settings
      </button>
      {open && (
        <div className="mt-2 space-y-2 pl-1">
          <SettingsForm
            token={token}
            organizationId={organizationId}
            connectionId={placement.connection_id}
            idPrefix={`deployment-${item.id}-${placement.connection_id}`}
            spec={spec}
            values={values}
            disabled={disabled}
            onChange={(name, v) => setValues((s) => ({ ...s, [name]: v }))}
          />
          <button
            type="button"
            disabled={updateM.isPending || disabled || errors.length > 0}
            onClick={() =>
              updateM.mutate({
                organizationId,
                diagramId,
                deploymentId: item.id,
                data: { placements: withPlacementSettings(item.placements ?? [], placement.connection_id, cleanValues(spec, values)) },
              })
            }
            className="rounded-md bg-slate-700 px-2.5 py-1 text-xs text-white hover:bg-slate-600 disabled:opacity-50">
            {updateM.isPending ? "Saving…" : "Save settings"}
          </button>
          {updateM.isError && (
            <p role="alert" className="text-xs text-red-300">
              {adapterErrorMessage(updateM.error)}
            </p>
          )}
        </div>
      )}
    </div>
  );
}
