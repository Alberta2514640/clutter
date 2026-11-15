"use client";

import { adapterErrorMessage } from "@/lib/adapterApi";
import { blockFields, blockTitle as titleOf, defaultValues, edgeAllowed, paletteBlocks, valuesErrors } from "@/lib/features/catalog/forms";
import { useCatalog } from "@/lib/features/catalog/hooks";
import { useConnections } from "@/lib/features/connections/hooks";
import {
  BLOCK_NODE_TYPE,
  defaultBlockName,
  fromBlocksDocument,
  newBlockId,
  toBlocksDocument,
  type BlockEdge,
  type BlockNode as BlockNodeType,
  type BlocksDocument,
} from "@/lib/features/diagram/blocksDocument";
import { useSaveBlocksDiagram } from "@/lib/features/diagram/hooks";
import { resourceNameError } from "@/lib/features/diagram/resourceName";
import { saveStatus } from "@/lib/features/diagram/saveStatus";
import type { Diagram } from "@/lib/features/diagram/types";
import type { BlockRef } from "@/lib/features/deployments/compatibility";
import { useDiagramDeployments, useInvalidateDeploymentChecks } from "@/lib/features/deployments/hooks";
import { BLOCKS_VIEW, nodeLook, resolveView, viewClouds, viewSummary } from "@/lib/features/deployments/views";
import { useOrganizations } from "@/lib/features/organization/hooks";
import { useProjectOrganization } from "@/lib/features/projects/hooks";
import { useMe } from "@/lib/features/user/hooks";
import {
  addEdge,
  applyEdgeChanges,
  applyNodeChanges,
  Background,
  BackgroundVariant,
  Controls,
  MarkerType,
  ReactFlow,
  useReactFlow,
  type Connection,
  type DefaultEdgeOptions,
  type EdgeChange,
  type EdgeTypes,
  type IsValidConnection,
  type NodeChange,
  type NodeTypes,
} from "@xyflow/react";
import { ArrowLeft, Check, Loader2, Save } from "lucide-react";
import { useRouter } from "next/navigation";
import { useCallback, useMemo, useRef, useState } from "react";
import BlockForm from "./BlockForm";
import BlockNode, { BlockLooks } from "./BlockNode";
import BlockPalette, { BLOCK_DND_MIME } from "./BlockPalette";
import DeployPanel from "./DeployPanel";
import FloatingEdge, { FLOATING_EDGE_TYPE } from "./FloatingEdge";
import { LEAVE_PROMPT, useLeaveGuard } from "./useLeaveGuard";
import ViewSwitch from "./ViewSwitch";

const nodeTypes: NodeTypes = { [BLOCK_NODE_TYPE]: BlockNode };
const edgeTypes: EdgeTypes = { [FLOATING_EDGE_TYPE]: FloatingEdge };
// Applied to every edge as it is drawn: saved edges carry no type or marker.
// The arrow shows which way an edge reads ("Firewall to VM").
const defaultEdgeOptions: DefaultEdgeOptions = {
  type: FLOATING_EDGE_TYPE,
  markerEnd: { type: MarkerType.ArrowClosed, width: 18, height: 18 },
};

// The "View as" choice is remembered per viewer and diagram, in this browser
// only. Storage can be blocked (a private window); the choice then lasts the visit.
const viewStorageKey = (diagramId: string) => `clutter.blocks-view.${diagramId}`;

function readViewChoice(diagramId: string): string {
  try {
    return window.localStorage.getItem(viewStorageKey(diagramId)) ?? BLOCKS_VIEW;
  } catch {
    return BLOCKS_VIEW;
  }
}

function writeViewChoice(diagramId: string, view: string) {
  try {
    window.localStorage.setItem(viewStorageKey(diagramId), view);
  } catch {
    // Storage is blocked; the choice lasts this visit.
  }
}

// A diagram's name is capped by A's diagrams table.
const DIAGRAM_NAME_MAX = 32;

type Snapshot = { name: string; doc: BlocksDocument };
const same = (a: Snapshot, b: Snapshot) => JSON.stringify(a) === JSON.stringify(b);

// A's diagram API answers an error with a JSON body holding a message.
function saveErrorMessage(err: unknown): string {
  const raw = err instanceof Error ? err.message : "";
  try {
    const parsed = JSON.parse(raw) as { message?: unknown };
    if (typeof parsed.message === "string" && parsed.message) return parsed.message;
  } catch {
    // Not JSON: the request itself failed.
  }
  return raw || "The save failed.";
}

// B's editor: building blocks from the manifests, saved through A's diagram
// API as a building-block document, deployed through B per linked cloud.
export default function BlocksEditor({ projectId, diagram }: { projectId: string; diagram: Diagram }) {
  const router = useRouter();
  const meQ = useMe();
  const token = meQ.data?.token ?? null;
  const orgsQ = useOrganizations(token);
  const orgIds = useMemo(() => orgsQ.data?.map((o) => o.id) ?? null, [orgsQ.data]);
  // Every B call is checked against the project's own organization.
  const orgId = useProjectOrganization(token, projectId, orgIds).data ?? null;

  const catalogQ = useCatalog(token);
  const catalog = useMemo(() => catalogQ.data ?? [], [catalogQ.data]);
  const palette = useMemo(() => paletteBlocks(catalog), [catalog]);
  const blockTitle = useCallback((kind: string) => titleOf(catalog, kind), [catalog]);

  // The clouds "View as" offers: those this diagram deploys to or the
  // organization connects to. The deploy panel reads the same two queries.
  const deploymentsQ = useDiagramDeployments(token, orgId, diagram.id);
  const connectionsQ = useConnections(token, orgId);
  const views = useMemo(
    () => viewClouds(catalog, deploymentsQ.data ?? [], connectionsQ.data ?? []),
    [catalog, deploymentsQ.data, connectionsQ.data],
  );
  const [viewChoice, setViewChoice] = useState(() => readViewChoice(diagram.id));
  const view = resolveView(viewChoice, views);
  const currentView = views.find((v) => v.connector === view) ?? null;
  const chooseView = (v: string) => {
    setViewChoice(v);
    writeViewChoice(diagram.id, v);
  };
  const lookFor = useCallback(
    (block: { id: string; kind: string }) => nodeLook(view, views, block, blockTitle(block.kind)),
    [view, views, blockTitle],
  );

  const saveM = useSaveBlocksDiagram(token);
  const invalidateChecks = useInvalidateDeploymentChecks();
  const { screenToFlowPosition } = useReactFlow();
  const canvasRef = useRef<HTMLDivElement>(null);

  const [initial] = useState(() => fromBlocksDocument(diagram.data));
  const [nodes, setNodes] = useState<BlockNodeType[]>(initial.nodes);
  const [edges, setEdges] = useState<BlockEdge[]>(initial.edges);
  const [name, setName] = useState(diagram.name ?? "");
  const [saved, setSaved] = useState<Snapshot>(() => ({ name: diagram.name ?? "", doc: toBlocksDocument(initial.nodes, initial.edges) }));
  // A diagram with no blocks yet opens on the Block tab: there is nothing to
  // deploy until something is drawn.
  const [tab, setTab] = useState<"block" | "deploy">(() => (initial.nodes.length > 0 ? "deploy" : "block"));
  const doc = toBlocksDocument(nodes, edges);
  const dirty = !same({ name, doc }, saved);
  const status = saveStatus({ dirty, saving: saveM.isPending });
  useLeaveGuard(dirty);
  const selected = nodes.find((n) => n.selected) ?? null;

  const savedBlocks: BlockRef[] = useMemo(
    () => saved.doc.nodes.map((n) => ({ id: n.id, kind: n.type, name: n.data.name })),
    [saved],
  );

  const saveProblem = (() => {
    if (!name.trim()) return "Give the diagram a name.";
    for (const n of nodes) {
      const label = n.data.name || blockTitle(n.data.kind);
      if (!n.data.name.trim()) return `${label}: every block needs a name.`;
      const nameErr = resourceNameError(n.data.name);
      if (nameErr) return `${label}: ${nameErr}`;
      const fieldErr = valuesErrors(blockFields(catalog, n.data.kind), n.data.fields)[0];
      if (fieldErr) return `${label}: ${fieldErr.message}`;
    }
    return null;
  })();

  const addBlock = useCallback(
    (kind: string, position: { x: number; y: number }) => {
      setNodes((cur) => [
        ...cur.map((n) => ({ ...n, selected: false })),
        {
          id: newBlockId(),
          type: BLOCK_NODE_TYPE,
          position,
          selected: true,
          data: {
            kind,
            name: defaultBlockName(kind, cur.map((n) => n.data.name)),
            fields: defaultValues(blockFields(catalog, kind)),
          },
        },
      ]);
      setTab("block");
    },
    [catalog],
  );

  const addAtCenter = (kind: string) => {
    const rect = canvasRef.current?.getBoundingClientRect();
    const offset = (nodes.length % 8) * 30;
    const center = rect ? { x: rect.left + rect.width / 2 - 75 + offset, y: rect.top + rect.height / 3 + offset } : { x: 200 + offset, y: 200 + offset };
    addBlock(kind, screenToFlowPosition(center));
  };

  const onDrop = (e: React.DragEvent) => {
    e.preventDefault();
    const kind = e.dataTransfer.getData(BLOCK_DND_MIME);
    if (!kind || !palette.some((b) => b.kind === kind)) return;
    addBlock(kind, screenToFlowPosition({ x: e.clientX, y: e.clientY }));
  };

  const onNodesChange = useCallback((changes: NodeChange<BlockNodeType>[]) => setNodes((cur) => applyNodeChanges(changes, cur)), []);
  const onEdgesChange = useCallback((changes: EdgeChange<BlockEdge>[]) => setEdges((cur) => applyEdgeChanges(changes, cur)), []);
  const onConnect = useCallback((c: Connection) => setEdges((cur) => addEdge({ ...c, id: newBlockId() }, cur)), []);

  // An edge is allowed when some cloud can wire its two kinds (the manifests'
  // edge lists) and it is not already drawn.
  const isValidConnection: IsValidConnection<BlockEdge> = (c) => {
    if (!c.source || !c.target || c.source === c.target) return false;
    const s = nodes.find((n) => n.id === c.source);
    const t = nodes.find((n) => n.id === c.target);
    if (!s || !t) return false;
    if (edges.some((e) => e.source === c.source && e.target === c.target)) return false;
    return edgeAllowed(catalog, s.data.kind, t.data.kind);
  };

  const patchNode = (id: string, patch: { name?: string; fields?: Record<string, unknown> }) =>
    setNodes((cur) => cur.map((n) => (n.id === id ? { ...n, data: { ...n.data, ...patch } } : n)));

  const removeNode = (id: string) => {
    setNodes((cur) => cur.filter((n) => n.id !== id));
    setEdges((cur) => cur.filter((e) => e.source !== id && e.target !== id));
  };

  const onSave = () => {
    if (saveProblem || !token) return;
    const trimmed = name.trim();
    const document = toBlocksDocument(nodes, edges);
    saveM.mutate(
      { projectId, diagramId: diagram.id, name: trimmed, document },
      {
        onSuccess: () => {
          setName(trimmed);
          setSaved({ name: trimmed, doc: document });
          invalidateChecks();
        },
      },
    );
  };

  const onBack = () => {
    if (dirty && !window.confirm(LEAVE_PROMPT)) return;
    router.push(`/projects/${projectId}/diagrams`);
  };

  return (
    <div className="flex h-screen w-screen flex-col overflow-hidden bg-slate-950 text-white">
      {/* Top bar */}
      <header className="flex items-center gap-3 border-b border-slate-800 px-4 py-2">
        <button type="button" onClick={onBack} aria-label="Back to the project" className="rounded p-1.5 text-slate-300 hover:bg-slate-800 hover:text-white">
          <ArrowLeft className="h-5 w-5" aria-hidden />
        </button>
        <input
          aria-label="Diagram name"
          value={name}
          maxLength={DIAGRAM_NAME_MAX}
          onChange={(e) => setName(e.target.value)}
          className="w-64 rounded-md border border-transparent bg-transparent px-2 py-1 text-base font-semibold text-white outline-none hover:border-slate-700 focus:border-teal-500/60"
        />
        <span className="shrink-0 rounded border border-slate-700 px-1.5 py-0.5 text-[11px] text-slate-400">Building blocks</span>
        <ViewSwitch views={views} value={view} onChange={chooseView} />
        <div className="ml-auto flex shrink-0 items-center gap-3">
          {saveM.isError && !saveM.isPending && (
            <span role="alert" className="max-w-md truncate text-xs text-red-300" title={saveErrorMessage(saveM.error)}>
              Not saved: {saveErrorMessage(saveM.error)}
            </span>
          )}
          {saveProblem && dirty && (
            <span className="max-w-md truncate text-xs text-amber-300" title={saveProblem}>
              {saveProblem}
            </span>
          )}
          <span
            role="status"
            data-testid="save-status"
            className={`flex items-center gap-1 whitespace-nowrap text-xs ${status.tone === "saved" ? "text-emerald-300" : status.tone === "unsaved" ? "text-amber-200" : "text-slate-400"}`}>
            {status.tone === "saved" && <Check className="h-3.5 w-3.5" aria-hidden />}
            {status.text}
          </span>
          <button
            type="button"
            onClick={onSave}
            disabled={!dirty || !!saveProblem || saveM.isPending || !token}
            className="inline-flex items-center gap-1.5 rounded-md bg-teal-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-teal-500 disabled:opacity-40">
            {saveM.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
            Save
          </button>
        </div>
      </header>

      <div className="flex min-h-0 flex-1">
        <BlockPalette
          blocks={palette}
          catalog={catalog}
          loading={catalogQ.isLoading}
          error={catalogQ.isError ? adapterErrorMessage(catalogQ.error) : null}
          onAdd={addAtCenter}
          blockTitle={blockTitle}
        />

        <div ref={canvasRef} className="relative min-w-0 flex-1" data-testid="blocks-canvas">
          <BlockLooks.Provider value={lookFor}>
            <ReactFlow<BlockNodeType, BlockEdge>
              colorMode="dark"
              nodes={nodes}
              edges={edges}
              nodeTypes={nodeTypes}
              edgeTypes={edgeTypes}
              defaultEdgeOptions={defaultEdgeOptions}
              onNodesChange={onNodesChange}
              onEdgesChange={onEdgesChange}
              onConnect={onConnect}
              isValidConnection={isValidConnection}
              onNodeClick={() => setTab("block")}
              onDragOver={(e) => {
                e.preventDefault();
                e.dataTransfer.dropEffect = "move";
              }}
              onDrop={onDrop}
              snapToGrid
              snapGrid={[20, 20]}
              fitView={nodes.length > 0}
              fitViewOptions={{ maxZoom: 1 }}>
              <Background variant={BackgroundVariant.Dots} gap={20} size={1.5} />
              <Controls showInteractive={false} />
            </ReactFlow>
          </BlockLooks.Provider>
          {currentView && (
            <p data-testid="view-summary" className="pointer-events-none absolute left-3 top-3 z-10 max-w-[calc(100%-1.5rem)] truncate rounded-md border border-slate-800 bg-slate-900/90 px-3 py-1.5 text-xs text-slate-300">
              {viewSummary(
                currentView,
                nodes.map((n) => ({ id: n.id, kind: n.data.kind })),
                blockTitle,
              )}
            </p>
          )}
          {nodes.length === 0 && (
            <div data-testid="canvas-hint" className="pointer-events-none absolute inset-0 flex items-center justify-center p-6">
              <div className="max-w-sm rounded-lg border border-slate-800 bg-slate-900/90 px-5 py-4 text-sm text-slate-300 shadow-lg">
                <h2 className="font-semibold text-white">Start your diagram</h2>
                <ol className="mt-2 list-decimal space-y-1 pl-5">
                  <li>Drag a building block from the left onto the canvas, or click it.</li>
                  <li>Link two blocks: drag from the right dot of one block to the left dot of another.</li>
                  <li>Save, then open the Deploy tab to build it in your cloud.</li>
                </ol>
              </div>
            </div>
          )}
        </div>

        <aside aria-label="Block and deployments" className="flex w-[380px] shrink-0 flex-col border-l border-slate-800 bg-slate-950">
          <div role="tablist" aria-label="Block and deployments" className="flex border-b border-slate-800">
            {(["block", "deploy"] as const).map((t) => (
              <button
                key={t}
                id={`side-tab-${t}`}
                type="button"
                role="tab"
                aria-selected={tab === t}
                aria-controls="side-tab-panel"
                onClick={() => setTab(t)}
                className={`flex-1 px-4 py-2.5 text-sm ${tab === t ? "border-b-2 border-teal-400 text-white" : "text-slate-400 hover:text-white"}`}>
                {t === "block" ? "Block" : "Deploy"}
              </button>
            ))}
          </div>
          <div id="side-tab-panel" role="tabpanel" aria-labelledby={`side-tab-${tab}`} className="min-h-0 flex-1 overflow-y-auto p-4">
            {tab === "block" ? (
              selected ? (
                <BlockForm
                  key={selected.id}
                  node={selected}
                  catalog={catalog}
                  title={blockTitle(selected.data.kind)}
                  disabled={false}
                  onChange={(patch) => patchNode(selected.id, patch)}
                  onRemove={() => removeNode(selected.id)}
                />
              ) : (
                <p className="text-sm text-slate-400">
                  {nodes.length === 0 ? "Add a building block from the left to start." : "Select a block to name it and set its fields."}
                </p>
              )
            ) : (
              <DeployPanel
                token={token}
                organizationId={orgId}
                diagramId={diagram.id}
                catalog={catalog}
                savedBlocks={savedBlocks}
                blockTitle={blockTitle}
                dirty={dirty}
              />
            )}
          </div>
        </aside>
      </div>
    </div>
  );
}
