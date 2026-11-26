"use client";

import CloudLogo from "@/components/common/CloudLogo";
import ErrorState from "@/components/common/ErrorState";
import StatusDot from "@/components/common/StatusDot";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { cloudLookFor, kindBadgeClass } from "@/lib/features/catalog/display";
import type { CatalogEntry } from "@/lib/features/catalog/types";
import type { BlockEdge, BlockNode } from "@/lib/features/diagram/blocksDocument";
import { DEPLOYMENT_STATE, deploymentState } from "@/lib/features/deployments/runState";
import { changesNotDeployed, type DeploymentListItem } from "@/lib/features/deployments/types";
import { Background, BackgroundVariant, ReactFlow, useReactFlow } from "@xyflow/react";
import { ArrowLeft, Maximize, Monitor, ZoomIn } from "lucide-react";
import Link from "next/link";
import { useState } from "react";
import { canvasStyle, defaultEdgeOptions, edgeTypes, nodeTypes } from "./canvasSetup";
import KindIcon from "./KindIcon";

type Props = {
  name: string;
  projectName: string | null;
  projectHref: string;
  nodes: BlockNode[];
  edges: BlockEdge[];
  colorMode: "light" | "dark";
  deployments: DeploymentListItem[];
  deploymentsLoading: boolean;
  deploymentsError: boolean;
  onRetryDeployments: () => void;
  catalog: CatalogEntry[];
  blockTitle: (kind: string) => string;
};

// The zoom the phone canvas opens at, at least: block names stay readable.
const READABLE_ZOOM = 0.6;

// On a phone the editor is a read-only view: the canvas at a zoom where names
// can be read (with a button to see the whole diagram), the blocks as a list,
// and each deployment with where it stands and whether it has changes not
// deployed yet. Editing needs a wider screen; the canvas keeps any unsaved
// work, so turning a tablet sideways brings it back.
export default function ReadOnlyView({
  name,
  projectName,
  projectHref,
  nodes,
  edges,
  colorMode,
  deployments,
  deploymentsLoading,
  deploymentsError,
  onRetryDeployments,
  catalog,
  blockTitle,
}: Props) {
  const { fitView } = useReactFlow();
  const [whole, setWhole] = useState(false);
  return (
    <div data-testid="narrow-screen" className="bg-background text-foreground min-h-dvh w-full overflow-y-auto">
      <header className="bg-background sticky top-0 z-20 flex h-14 items-center gap-2 border-b px-4">
        <Button asChild variant="ghost" size="icon-sm">
          <Link href={projectHref} aria-label="Back to the project">
            <ArrowLeft aria-hidden />
          </Link>
        </Button>
        <div className="min-w-0">
          <p className="text-muted-foreground truncate text-xs">{projectName ?? "Project"}</p>
          <h1 className="truncate text-sm font-semibold">{name || "Untitled diagram"}</h1>
        </div>
      </header>

      <p className="bg-muted/60 text-muted-foreground flex items-start gap-2 border-b px-4 py-2.5 text-xs">
        <Monitor className="mt-px size-3.5 shrink-0" aria-hidden />
        You are viewing this diagram. To edit or deploy it, open it on a screen at least 1024 pixels wide, or turn a tablet sideways.
      </p>

      <div data-testid="blocks-canvas" className="relative h-[55dvh] border-b">
        {nodes.length === 0 ? (
          <p className="text-muted-foreground grid h-full place-items-center p-6 text-center text-sm">This diagram has no blocks yet.</p>
        ) : (
          <>
            {/* It opens at a zoom where names can be read, even when that
                crops a large diagram; drag to move, or show the whole of it. */}
            <ReactFlow<BlockNode, BlockEdge>
              colorMode={colorMode}
              style={canvasStyle}
              nodes={nodes}
              edges={edges}
              nodeTypes={nodeTypes}
              edgeTypes={edgeTypes}
              defaultEdgeOptions={defaultEdgeOptions}
              nodesDraggable={false}
              nodesConnectable={false}
              elementsSelectable={false}
              deleteKeyCode={null}
              fitView
              fitViewOptions={{ padding: 0.15, minZoom: READABLE_ZOOM }}
              minZoom={0.2}>
              <Background variant={BackgroundVariant.Dots} gap={16} size={1.2} color="var(--dot-grid)" />
            </ReactFlow>
            <div className="pointer-events-none absolute inset-x-3 bottom-3 z-10 flex items-center justify-between gap-2">
              <span className="surface-raised text-muted-foreground rounded-full border px-2.5 py-1 text-[11px]">Drag to move around.</span>
              <Button
                type="button"
                size="xs"
                variant="outline"
                className="surface-raised pointer-events-auto"
                aria-pressed={whole}
                onClick={() => {
                  const next = !whole;
                  setWhole(next);
                  void fitView({ padding: 0.15, minZoom: next ? 0.2 : READABLE_ZOOM, duration: 200 });
                }}>
                {whole ? <ZoomIn aria-hidden /> : <Maximize aria-hidden />}
                {whole ? "Zoom to read" : "View full diagram"}
              </Button>
            </div>
          </>
        )}
      </div>

      <section aria-labelledby="ro-deployments" className="space-y-3 px-4 py-5">
        <h2 id="ro-deployments" className="text-sm font-semibold">
          Deployments
        </h2>
        {deploymentsLoading ? (
          <div role="status" className="space-y-2">
            <span className="sr-only">Loading deployments…</span>
            <Skeleton className="h-14" />
          </div>
        ) : deploymentsError ? (
          <ErrorState variant="inline" description="Could not load this diagram's deployments." onRetry={onRetryDeployments} />
        ) : deployments.length === 0 ? (
          <p className="text-muted-foreground text-sm">Not deployed anywhere yet.</p>
        ) : (
          <ul className="space-y-2">
            {deployments.map((d) => {
              const state = DEPLOYMENT_STATE[deploymentState(d)];
              const look = cloudLookFor(catalog, d.connector);
              return (
                <li key={d.id} data-testid="deployment-row" className="surface-card flex items-center gap-3 px-3 py-2.5">
                  <CloudLogo connector={look.connector} shortName={look.shortName} src={look.logoSrc} size={22} />
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-sm font-medium">{d.connection_name}</span>
                    <span className="text-muted-foreground block truncate text-xs">{look.title}</span>
                  </span>
                  <span className="flex shrink-0 flex-col items-end gap-1">
                    <Badge variant={state.tone}>
                      <StatusDot tone={state.tone} pulse={state.running} />
                      {state.label}
                    </Badge>
                    {changesNotDeployed(d) && (
                      <Badge variant="warning" data-testid="changes-not-deployed">
                        Changes not deployed yet
                      </Badge>
                    )}
                  </span>
                </li>
              );
            })}
          </ul>
        )}
      </section>

      <section aria-labelledby="ro-blocks" className="space-y-3 border-t px-4 py-5">
        <h2 id="ro-blocks" className="text-sm font-semibold">
          Blocks ({nodes.length})
        </h2>
        <ul className="divide-y rounded-lg border">
          {nodes.map((n) => (
            <li key={n.id} className="flex items-center gap-3 px-3 py-2.5">
              <span className={`${kindBadgeClass(n.data.kind)} flex size-7 shrink-0 items-center justify-center rounded-md`}>
                <KindIcon kind={n.data.kind} className="size-4" />
              </span>
              <span className="min-w-0 flex-1 truncate font-mono text-sm">{n.data.name || "(no name)"}</span>
              <span className="text-muted-foreground shrink-0 text-xs">{blockTitle(n.data.kind)}</span>
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}
