"use client";

import { adapterErrorMessage } from "@/lib/adapterApi";
import { blockFields, blockTitle as titleOf, defaultValues, paletteBlocks, valuesErrors } from "@/lib/features/catalog/forms";
import { useCatalog } from "@/lib/features/catalog/hooks";
import { allLinkCards, linkCard, linkCheck, manifestsInPlay } from "@/lib/features/catalog/links";
import { useConnections } from "@/lib/features/connections/hooks";
import {
  BLOCK_NODE_TYPE,
  BOX,
  boxAt,
  defaultBlockName,
  detachFrom,
  fromBlocksDocument,
  NETWORK_KIND,
  NETWORK_NODE_TYPE,
  newBlockId,
  nodeTypeFor,
  parentsFirst,
  placeIn,
  toBlocksDocument,
  VM_KIND,
  type BlockEdge,
  type BlockNode as BlockNodeType,
  type BlocksDocument,
} from "@/lib/features/diagram/blocksDocument";
import { DIAGRAM_NAME_MAX } from "@/lib/features/diagram/diagramName";
import { useSaveBlocksDiagram } from "@/lib/features/diagram/hooks";
import { resourceNameError } from "@/lib/features/diagram/resourceName";
import { saveStatus } from "@/lib/features/diagram/saveStatus";
import { diagramSummary } from "@/lib/features/diagram/summary";
import type { Diagram } from "@/lib/features/diagram/types";
import { problemsByBlock, type BlockRef } from "@/lib/features/deployments/compatibility";
import { useCompatibility, useDiagramDeployments, useInvalidateDeploymentChecks } from "@/lib/features/deployments/hooks";
import { servesWeb } from "@/lib/features/deployments/results";
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
  useConnection,
  useReactFlow,
  type Connection,
  type DefaultEdgeOptions,
  type EdgeChange,
  type EdgeTypes,
  type IsValidConnection,
  type NodeChange,
  type NodeTypes,
  type OnBeforeDelete,
  type OnConnectEnd,
  type XYPosition,
} from "@xyflow/react";
import { ArrowLeft, Check, Loader2, Save } from "lucide-react";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import BlockForm from "./BlockForm";
import BlockNode from "./BlockNode";
import BlockPalette, { BLOCK_DND_MIME } from "./BlockPalette";
import { CanvasContext, type Canvas, type EdgeCard } from "./canvasContext";
import DeployPanel from "./DeployPanel";
import FloatingEdge, { FLOATING_EDGE_TYPE } from "./FloatingEdge";
import NetworkBox from "./NetworkBox";
import SummaryPill from "./SummaryPill";
import { LEAVE_PROMPT, useLeaveGuard } from "./useLeaveGuard";
import ViewSwitch from "./ViewSwitch";

const nodeTypes: NodeTypes = { [BLOCK_NODE_TYPE]: BlockNode, [NETWORK_NODE_TYPE]: NetworkBox };
const edgeTypes: EdgeTypes = { [FLOATING_EDGE_TYPE]: FloatingEdge };
// Applied to every edge as it is drawn: saved edges carry no type or marker.
// The arrow shows which way an edge reads ("Firewall to VM").
const defaultEdgeOptions: DefaultEdgeOptions = {
  type: FLOATING_EDGE_TYPE,
  markerEnd: { type: MarkerType.ArrowClosed, width: 18, height: 18 },
};

// The canvas's background, which a box's label sits on.
const CANVAS_BG = "#020617";

// How long a refused line's reason stays on screen.
const REFUSAL_MS = 3500;

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

// The middle of a block card whose top left corner is at p.
const cardCenter = (p: XYPosition, n?: BlockNodeType): XYPosition => ({
  x: p.x + (n?.measured?.width ?? BOX.blockW) / 2,
  y: p.y + (n?.measured?.height ?? BOX.blockH) / 2,
});

// B's editor: building blocks from the manifests, saved through A's diagram
// API as a building-block document, deployed through B per linked cloud.
// Networks are boxes and a VM inside one runs in it (ADR-025); every other
// link is a line, and while one is drawn only the blocks it may reach glow.
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
  // The clouds in play decide which lines may be drawn and what they mean.
  const manifests = useMemo(
    () => manifestsInPlay(catalog, currentView ? [currentView.connector] : views.map((v) => v.connector)),
    [catalog, currentView, views],
  );

  // Problem badges come from one deployment's compatibility check: the first
  // on the cloud being viewed, or the first of all.
  const deployments = deploymentsQ.data ?? [];
  const checked = (currentView && deployments.find((d) => d.connector === currentView.connector)) || deployments[0] || null;
  const compatQ = useCompatibility(token, orgId, checked?.id ?? null);
  const problems = useMemo(() => problemsByBlock(compatQ.data), [compatQ.data]);

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

  // The box a VM being dragged would drop into, and the kind a palette drag holds.
  const [dropTarget, setDropTarget] = useState<string | null>(null);
  const dragKind = useRef<string | null>(null);
  // Why the last line was refused, shown for a moment.
  const [refusal, setRefusal] = useState<string | null>(null);
  useEffect(() => {
    if (!refusal) return;
    const t = window.setTimeout(() => setRefusal(null), REFUSAL_MS);
    return () => window.clearTimeout(t);
  }, [refusal]);

  const savedBlocks: BlockRef[] = useMemo(
    () => saved.doc.nodes.map((n) => ({ id: n.id, kind: n.type, name: n.data.name })),
    [saved],
  );
  const currentBlocks: BlockRef[] = useMemo(() => nodes.map((n) => ({ id: n.id, kind: n.data.kind, name: n.data.name })), [nodes]);
  // The saved servers that are a website, for "What was built".
  const webServers = useMemo(
    () => servesWeb(saved.doc.nodes.map((n) => ({ id: n.id, kind: n.type, fields: n.data.fields })), saved.doc.edges),
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

  // addBlock adds a block with its top left corner at position on the canvas;
  // a VM dropped on a box goes into it.
  const addBlock = useCallback(
    (kind: string, position: XYPosition, box: string | null = null) => {
      const id = newBlockId();
      setNodes((cur) => {
        const node: BlockNodeType = {
          id,
          type: nodeTypeFor(kind),
          position,
          selected: true,
          data: {
            kind,
            name: defaultBlockName(kind, cur.map((n) => n.data.name)),
            fields: defaultValues(blockFields(catalog, kind)),
          },
          ...(kind === NETWORK_KIND ? { width: BOX.width, height: BOX.height } : {}),
        };
        const next = parentsFirst([...cur.map((n) => ({ ...n, selected: false })), node]);
        return kind === VM_KIND && box ? placeIn(next, id, box, position) : next;
      });
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

  const onDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
    const box = dragKind.current === VM_KIND ? boxAt(nodes, screenToFlowPosition({ x: e.clientX, y: e.clientY })) : null;
    if (box !== dropTarget) setDropTarget(box);
  };

  const onDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDropTarget(null);
    dragKind.current = null;
    const kind = e.dataTransfer.getData(BLOCK_DND_MIME);
    if (!kind || !palette.some((b) => b.kind === kind)) return;
    const at = screenToFlowPosition({ x: e.clientX, y: e.clientY });
    addBlock(kind, at, kind === VM_KIND ? boxAt(nodes, at) : null);
  };

  // While a VM is dragged on the canvas, the box under its middle lights up,
  // unless it is the box the VM is already in. Dropped, the VM moves into
  // that box, or out of its box onto the canvas.
  const boxUnder = useCallback(
    (n: BlockNodeType, all: BlockNodeType[]) => {
      const byId = new Map(all.map((x) => [x.id, x]));
      const parent = n.parentId ? byId.get(n.parentId) : undefined;
      const abs = parent ? { x: parent.position.x + n.position.x, y: parent.position.y + n.position.y } : n.position;
      return { abs, box: boxAt(all, cardCenter(abs, n)) };
    },
    [],
  );

  const onNodeDrag = (_e: React.MouseEvent, n: BlockNodeType) => {
    if (n.data.kind !== VM_KIND) return;
    const { box } = boxUnder(n, nodes);
    const next = box && box !== n.parentId ? box : null;
    if (next !== dropTarget) setDropTarget(next);
  };

  const onNodeDragStop = (_e: React.MouseEvent, _n: BlockNodeType, dragged: BlockNodeType[]) => {
    setDropTarget(null);
    setNodes((cur) => {
      let next = cur;
      for (const d of dragged) {
        if (d.data.kind !== VM_KIND) continue;
        const { abs, box } = boxUnder(d, next);
        if ((box ?? null) !== (d.parentId ?? null)) next = placeIn(next, d.id, box, abs);
      }
      return next;
    });
  };

  const onNodesChange = useCallback((changes: NodeChange<BlockNodeType>[]) => setNodes((cur) => applyNodeChanges(changes, cur)), []);
  const onEdgesChange = useCallback((changes: EdgeChange<BlockEdge>[]) => setEdges((cur) => applyEdgeChanges(changes, cur)), []);
  const onConnect = useCallback((c: Connection) => setEdges((cur) => addEdge({ ...c, id: newBlockId() }, cur)), []);

  // Removing a box keeps the VMs in it, on the canvas where they were; React
  // Flow would remove them with it.
  const onBeforeDelete: OnBeforeDelete<BlockNodeType, BlockEdge> = async ({ nodes: gone, edges: goneEdges }) => {
    const boxes = new Set(gone.filter((n) => n.type === NETWORK_NODE_TYPE).map((n) => n.id));
    const kept = new Set(gone.filter((n) => n.parentId && boxes.has(n.parentId) && !n.selected).map((n) => n.id));
    if (kept.size === 0) return true;
    setNodes((cur) => detachFrom(cur, boxes));
    const removed = gone.filter((n) => !kept.has(n.id));
    const removedIds = new Set(removed.map((n) => n.id));
    return { nodes: removed, edges: goneEdges.filter((e) => e.selected || removedIds.has(e.source) || removedIds.has(e.target)) };
  };

  // Which lines may be drawn: the manifests of the clouds in play decide.
  const kindById = useMemo(() => new Map(nodes.map((n) => [n.id, n.data.kind])), [nodes]);
  const check = useCallback(
    (from: string, to: string) => {
      const fk = kindById.get(from);
      const tk = kindById.get(to);
      if (!fk || !tk) return { ok: false as const, reason: "" };
      return linkCheck(manifests, { id: from, kind: fk }, { id: to, kind: tk }, edges, (id) => kindById.get(id), blockTitle);
    },
    [manifests, kindById, edges, blockTitle],
  );
  const connectingFrom = useConnection((c) => (c.inProgress ? c.fromNode.id : null));

  const isValidConnection: IsValidConnection<BlockEdge> = (c) => !!c.source && !!c.target && check(c.source, c.target).ok;

  const onConnectEnd: OnConnectEnd = (_e, state) => {
    if (!state.fromNode || !state.toNode || state.isValid !== false || state.toNode.id === state.fromNode.id) return;
    const r = check(state.fromNode.id, state.toNode.id);
    if (!r.ok && r.reason) setRefusal(r.reason);
  };

  const patchNode = (id: string, patch: { name?: string; fields?: Record<string, unknown> }) =>
    setNodes((cur) => cur.map((n) => (n.id === id ? { ...n, data: { ...n.data, ...patch } } : n)));

  const removeNode = (id: string) => {
    setNodes((cur) => detachFrom(cur, new Set([id])).filter((n) => n.id !== id));
    setEdges((cur) => cur.filter((e) => e.source !== id && e.target !== id));
  };

  // save stores the canvas; it answers false when it could not (the header
  // says why: the save problem or the failed save).
  const save = async (): Promise<boolean> => {
    if (saveProblem || !token) return false;
    const trimmed = name.trim();
    const document = toBlocksDocument(nodes, edges);
    try {
      await saveM.mutateAsync({ projectId, diagramId: diagram.id, name: trimmed, document });
    } catch {
      return false;
    }
    setName(trimmed);
    setSaved({ name: trimmed, doc: document });
    invalidateChecks();
    return true;
  };
  const onSave = () => void save();

  const onBack = () => {
    if (dirty && !window.confirm(LEAVE_PROMPT)) return;
    router.push(`/projects/${projectId}/diagrams`);
  };

  const nameOf = useCallback((id: string) => nodes.find((n) => n.id === id)?.data.name || "(no name)", [nodes]);

  const canvas: Canvas = {
    look: (block) => nodeLook(view, views, block, blockTitle(block.kind)),
    dropTarget,
    linkState: (id) => {
      if (!connectingFrom || id === connectingFrom) return null;
      return check(connectingFrom, id).ok ? "valid" : "invalid";
    },
    problems: (id) => problems.get(id) ?? [],
    fieldsOf: (kind) => blockFields(catalog, kind),
    setField: (id, field, value) =>
      setNodes((cur) => cur.map((n) => (n.id === id ? { ...n, data: { ...n.data, fields: { ...n.data.fields, [field]: value } } } : n))),
    edgeCard: (source, target) => {
      const sk = kindById.get(source);
      const tk = kindById.get(target);
      const c = sk && tk ? linkCard(manifests, catalog, sk, tk) : null;
      return c ? { ...c, sourceName: nameOf(source), targetName: nameOf(target) } : null;
    },
  };

  const links: EdgeCard[] = useMemo(
    () => allLinkCards(manifests).map((c) => ({ ...c, sourceName: titleOf(catalog, c.source), targetName: titleOf(catalog, c.target) })),
    [manifests, catalog],
  );
  const summary = diagramSummary(
    nodes.map((n) => ({ kind: n.data.kind, name: n.data.name })),
    blockTitle,
  );

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
          loading={catalogQ.isLoading}
          error={catalogQ.isError ? adapterErrorMessage(catalogQ.error) : null}
          onAdd={addAtCenter}
          onDragKind={(k) => {
            dragKind.current = k;
            if (!k) setDropTarget(null);
          }}
          links={links}
        />

        <div ref={canvasRef} className="relative min-w-0 flex-1" data-testid="blocks-canvas">
          <CanvasContext.Provider value={canvas}>
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
              onConnectEnd={onConnectEnd}
              isValidConnection={isValidConnection}
              onBeforeDelete={onBeforeDelete}
              onNodeClick={() => setTab("block")}
              onNodeDrag={onNodeDrag}
              onNodeDragStop={onNodeDragStop}
              onDragOver={onDragOver}
              onDragLeave={(e) => {
                if (!e.currentTarget.contains(e.relatedTarget as Node | null)) setDropTarget(null);
              }}
              onDrop={onDrop}
              elevateNodesOnSelect={false}
              snapToGrid
              snapGrid={[20, 20]}
              fitView={nodes.length > 0}
              fitViewOptions={{ maxZoom: 1 }}>
              <Background variant={BackgroundVariant.Dots} gap={20} size={1.5} bgColor={CANVAS_BG} color="#1e293b" />
              <Controls showInteractive={false} />
            </ReactFlow>
          </CanvasContext.Provider>
          {currentView && (
            <p data-testid="view-summary" className="pointer-events-none absolute left-3 top-3 z-10 max-w-[calc(100%-1.5rem)] truncate rounded-md border border-slate-800 bg-slate-900/90 px-3 py-1.5 text-xs text-slate-300">
              {viewSummary(
                currentView,
                nodes.map((n) => ({ id: n.id, kind: n.data.kind })),
                blockTitle,
              )}
            </p>
          )}
          {refusal && (
            <p
              role="status"
              data-testid="link-refused"
              className="pointer-events-none absolute left-1/2 top-3 z-20 -translate-x-1/2 rounded-full border border-amber-700/70 bg-slate-900/95 px-3.5 py-1.5 text-xs text-amber-200 shadow-lg">
              {refusal}
            </p>
          )}
          {nodes.length === 0 ? (
            <div data-testid="canvas-hint" className="pointer-events-none absolute inset-0 flex items-center justify-center p-6">
              <p className="rounded-full border border-slate-800 bg-slate-900/90 px-4 py-2 text-sm text-slate-300 shadow-lg">
                Drag a block here to start. Drop VMs into a network box.
              </p>
            </div>
          ) : (
            <SummaryPill summary={summary} />
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
                <p className="text-sm text-slate-400">{nodes.length === 0 ? "Add a building block from the left to start." : "Select a block to name it and set its fields."}</p>
              )
            ) : (
              <DeployPanel
                token={token}
                organizationId={orgId}
                diagramId={diagram.id}
                catalog={catalog}
                savedBlocks={savedBlocks}
                currentBlocks={currentBlocks}
                webServers={webServers}
                blockTitle={blockTitle}
                dirty={dirty}
                save={save}
              />
            )}
          </div>
        </aside>
      </div>
    </div>
  );
}
