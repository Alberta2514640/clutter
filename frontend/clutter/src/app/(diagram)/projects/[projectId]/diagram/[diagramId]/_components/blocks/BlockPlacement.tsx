"use client";

import { adapterErrorMessage } from "@/lib/adapterApi";
import { cleanValues, cloudOffers, defaultValues, entryFor, settingsFor, valuesErrors } from "@/lib/features/catalog/forms";
import type { CatalogEntry } from "@/lib/features/catalog/types";
import type { Connection } from "@/lib/features/connections/types";
import type { BlockRef } from "@/lib/features/deployments/compatibility";
import { useUpdateDeployment } from "@/lib/features/deployments/hooks";
import { blockCloud, cloudBlocks, cloudChoices, placeBlock, placementReason, withPlacementSettings } from "@/lib/features/deployments/placements";
import type { DeploymentListItem, Placement } from "@/lib/features/deployments/types";
import { Button } from "@/components/ui/button";
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
  // Open until the user closes it: the diagram's blocks need two clouds.
  defaultOpen: boolean;
};

// BlockPlacement is a deployment's cloud per block (ADR-020): each
// block runs on the deployment's own connection unless it is placed on
// another of the organization's connections, which has its own settings. B
// checks that every connection is the organization's and that each edge can
// cross clouds; the compatibility line above says what it found. Moving a
// block to a cloud also puts it back in the deployment, so it is never left
// out of every cloud.
export default function BlockPlacement({ token, organizationId, diagramId, item, catalog, connections, blocks, blockTitle, running, defaultOpen }: Props) {
  const updateM = useUpdateDeployment(token);
  const [chosenOpen, setOpen] = useState<boolean | null>(null);
  const open = chosenOpen ?? defaultOpen;
  const [editing, setEditing] = useState(false);
  const placements: Placement[] = item.placements ?? [];
  const primary = connections.find((c) => c.id === item.connection_id);
  const choices = cloudChoices(primary, connections, (c) => !!entryFor(catalog, c), new Set(placements.map((p) => p.connection_id)));
  const reason = placementReason(item.status, running || updateM.isPending);
  // Each block is offered only the connections whose cloud can build it, and
  // the one it is on now.
  const choicesFor = (b: BlockRef, current: string) => choices.filter((c) => c.id === current || cloudOffers(catalog, c.connector, b.kind));
  const nameOf = (id: string) => connections.find((c) => c.id === id)?.name ?? placements.find((p) => p.connection_id === id)?.connection_name ?? item.connection_name;

  // With one usable cloud there is nothing to choose.
  if (choices.length < 2 && placements.length === 0) return null;

  const move = (blockId: string, connectionId: string) =>
    updateM.mutate({
      organizationId,
      diagramId,
      deploymentId: item.id,
      data: placeBlock(item.connection_id, placements, item.exclusions ?? [], blocks, blockId, connectionId),
    });

  return (
    <div>
      <button
        type="button"
        aria-expanded={open}
        onClick={() => setOpen(!open)}
        className="text-muted-foreground hover:text-foreground flex min-h-6 items-center gap-1 text-xs font-medium">
        {open ? <ChevronDown className="size-3.5" aria-hidden /> : <ChevronRight className="size-3.5" aria-hidden />}
        Where each block runs
        {placements.length > 0 && (
          <span className="text-muted-foreground font-normal">
            {" "}
            ({placements.length + 1} clouds)
          </span>
        )}
      </button>
      {open && (
        <div className="mt-2 space-y-2 pl-1" data-testid="block-placement">
          {blocks.length === 0 ? (
            <p className="text-muted-foreground text-xs">Save the diagram with blocks first.</p>
          ) : (
            // One compact row per block: its name and its cloud. "Change" turns
            // each cloud into a choice of the clouds that can build the block.
            <ul className="space-y-1 text-xs" aria-label="The cloud each block runs on">
              {blocks.map((b) => {
                const selectId = `${item.id}-cloud-${b.id}`;
                const current = blockCloud(item.connection_id, placements, b.id);
                return (
                  <li key={b.id} className="flex min-w-0 items-center justify-between gap-2">
                    <label htmlFor={editing ? selectId : undefined} className="min-w-0 truncate" title={`${b.name} (${blockTitle(b.kind)})`}>
                      {b.name}
                    </label>
                    {editing ? (
                      <select
                        id={selectId}
                        value={current}
                        onChange={(e) => move(b.id, e.target.value)}
                        className="border-input bg-background focus-visible:ring-ring/50 h-7 w-40 shrink-0 truncate rounded-md border px-1.5 text-xs outline-none focus-visible:ring-[3px]">
                        {choicesFor(b, current).map((c) => (
                          <option key={c.id} value={c.id}>
                            {c.name}
                          </option>
                        ))}
                      </select>
                    ) : (
                      <span data-testid="block-cloud" className="text-muted-foreground shrink-0 truncate">
                        {nameOf(current)}
                      </span>
                    )}
                  </li>
                );
              })}
            </ul>
          )}
          {blocks.length > 0 &&
            (reason ? (
              <p className="text-warning text-xs">{reason}</p>
            ) : (
              <Button type="button" variant="outline" size="xs" aria-expanded={editing} onClick={() => setEditing((e) => !e)}>
                {editing ? "Done" : "Change"}
              </Button>
            ))}
          {updateM.isPending && <p className="text-muted-foreground text-xs">Saving…</p>}
          {updateM.isError && (
            <p role="alert" className="text-danger text-xs">
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
              entry={entryFor(catalog, p.connector)}
              kinds={new Set(cloudBlocks(item.connection_id, placements, blocks, p.connection_id).map((b) => b.kind))}
              disabled={running}
            />
          ))}
        </div>
      )}
    </div>
  );
}

// PlacedCloudSettings edits one placed cloud's own settings (its image, its
// instance type), only those about the blocks placed there. They may change
// while the deployment is live.
function PlacedCloudSettings({
  token,
  organizationId,
  diagramId,
  item,
  placement,
  entry,
  kinds,
  disabled,
}: {
  token: string | null;
  organizationId: string;
  diagramId: string;
  item: DeploymentListItem;
  placement: Placement;
  entry: CatalogEntry | null;
  kinds: ReadonlySet<string>;
  disabled: boolean;
}) {
  const spec = entry?.manifest.settings ?? [];
  const shown = settingsFor(spec, kinds);
  const updateM = useUpdateDeployment(token);
  const [open, setOpen] = useState(false);
  const [values, setValues] = useState<Record<string, unknown>>(() => ({ ...defaultValues(spec), ...(placement.settings ?? {}) }));
  if (shown.length === 0) return null;
  const errors = valuesErrors(shown, values);
  return (
    <div>
      <button type="button" aria-expanded={open} onClick={() => setOpen((o) => !o)} className="text-muted-foreground hover:text-foreground flex min-h-6 items-center gap-1 text-xs">
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
            spec={shown}
            values={values}
            disabled={disabled}
            onChange={(name, v) => setValues((s) => ({ ...s, [name]: v }))}
          />
          <Button
            type="button"
            variant="secondary"
            size="xs"
            disabled={updateM.isPending || disabled || errors.length > 0}
            onClick={() =>
              updateM.mutate({
                organizationId,
                diagramId,
                deploymentId: item.id,
                data: { placements: withPlacementSettings(item.placements ?? [], placement.connection_id, cleanValues(spec, values)) },
              })
            }>
            {updateM.isPending ? "Saving…" : "Save settings"}
          </Button>
          {updateM.isError && (
            <p role="alert" className="text-danger text-xs">
              {adapterErrorMessage(updateM.error)}
            </p>
          )}
        </div>
      )}
    </div>
  );
}
