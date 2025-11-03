#!/usr/bin/env bash
set -euo pipefail

# Helper to write a file safely
mkfile () {
  local path="$1"
  mkdir -p "$(dirname "$path")"
  cat > "$path"
}

echo "==> Creating folders..."
mkdir -p \
  src/providers \
  src/lib/api \
  src/lib/state \
  src/lib/graph \
  src/lib/workers \
  src/lib/utils \
  src/components/shell \
  src/components/diagram \
  src/components/runs \
  src/components/common \
  src/app/projects \
  "src/app/projects/[pid]" \
  "src/app/projects/[pid]/diagrams/[did]" \
  "src/app/projects/[pid]/workspaces/[wid]" \
  "src/app/projects/[pid]/runs" \
  "src/app/projects/[pid]/settings" \
  src/tests \
  src/e2e

echo "==> Providers..."
mkfile src/providers/QueryProvider.tsx <<'TS'
"use client";
import React, { ReactNode, useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
export function QueryProvider({ children }: { children: ReactNode }) {
  const [client] = useState(() => new QueryClient({
    defaultOptions: { queries: { retry: 1, refetchOnWindowFocus: false, staleTime: 10_000 } }
  }));
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}
TS

mkfile src/providers/AuthProvider.tsx <<'TS'
"use client";
import React, { createContext, useContext, useEffect, useState } from "react";
type User = { id: string; name: string; role?: "owner"|"editor"|"viewer" } | null;
const Ctx = createContext<{ user: User; loading: boolean }>({ user: null, loading: true });
export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User>(null);
  const [loading, setLoading] = useState(true);
  useEffect(() => { setUser({ id: "me", name: "Nimna", role: "owner" }); setLoading(false); }, []);
  return <Ctx.Provider value={{ user, loading }}>{children}</Ctx.Provider>;
}
export const useAuth = () => useContext(Ctx);
TS

mkfile src/providers/YProvider.tsx <<'TS'
"use client";
import React, { createContext, useMemo, useEffect } from "react";
import * as Y from "yjs";
import { WebsocketProvider } from "y-websocket";
export const YCtx = createContext<{ doc: Y.Doc; provider: WebsocketProvider } | null>(null);
export function YProvider({ room, children }: { room: string; children: React.ReactNode }) {
  const doc = useMemo(() => new Y.Doc(), []);
  const provider = useMemo(
    () => new WebsocketProvider(process.env.NEXT_PUBLIC_YWS || "ws://localhost:1234", room, doc),
    [room, doc]
  );
  useEffect(() => () => provider.destroy(), [provider]);
  return <YCtx.Provider value={{ doc, provider }}>{children}</YCtx.Provider>;
}
TS

echo "==> lib/api, types, utils..."
mkfile src/lib/api/client.ts <<'TS'
export async function api<T>(url: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(url, {
    credentials: "include",
    headers: { "Content-Type": "application/json", ...(init.headers || {}) },
    ...init
  });
  if (!res.ok) throw Object.assign(new Error(res.statusText), { status: res.status, body: await res.text() });
  return res.status === 204 ? (undefined as T) : (await res.json() as T);
}
export const apiWithIfMatch = <B,R>(url:string, body:B, etag:string) =>
  api<R>(url, { method:"PATCH", body: JSON.stringify(body), headers: { "If-Match": etag }});
TS

mkfile src/lib/api/routes.ts <<'TS'
export const routes = {
  canvasBundle: (pid:string, cid:string) => `/api/projects/${pid}/canvases/${cid}/bundle`,
  node:        (pid:string, nid:string) => `/api/projects/${pid}/nodes/${nid}`,
  edge:        (pid:string, eid:string) => `/api/projects/${pid}/edges/${eid}`,
  runs:        (pid:string)             => `/api/projects/${pid}/runs`,
  run:         (pid:string, rid:string) => `/api/projects/${pid}/runs/${rid}`,
  exportZip:   (pid:string)             => `/api/projects/${pid}/export`,
  lockAcquire:                             `/api/locks`,
  lockRelease: (lid:string)             => `/api/locks/${lid}`,
};
TS

mkfile src/lib/queryKeys.ts <<'TS'
export const qk = {
  tenant: (tenantId: string) => ["tenant", tenantId] as const,
  projects: (tenantId: string) => ["projects", tenantId] as const,
  project: (projectId: string) => ["project", projectId] as const,
  canvasBundle: (projectId: string, canvasId: string) => ["canvasBundle", projectId, canvasId] as const,
  workspaces: (tenantId: string) => ["workspaces", tenantId] as const,
  workspace: (projectId: string, workspaceId: string) => ["workspace", projectId, workspaceId] as const,
  moduleSet: (projectId: string, moduleSetId: string) => ["moduleSet", projectId, moduleSetId] as const,
  varSets: (projectId: string, env?: string) => ["varSets", projectId, env] as const,
  runs: (projectId: string) => ["runs", projectId] as const,
  run: (projectId: string, runId: string) => ["run", projectId, runId] as const,
};
TS

mkfile src/lib/types.ts <<'TS'
export interface CanvasBundle { canvas: Canvas; nodes: Node[]; edges: Edge[]; etag?: string; version?: number; }
export interface Canvas { canvasId: string; name: string; uiLayout?: any; }
export interface Node { nodeId: string; canvasId: string; resourceType: string; spec: any; iac?: { moduleAlias: string; moduleSource?: string; version?: string; }; ui?: any; etag?: string; version?: number; }
export interface Edge { edgeId: string; fromNodeId: string; toNodeId: string; relation: string; props?: any; etag?: string; version?: number; }
export interface Workspace { workspaceId: string; name: string; accountRef: { tenantId: string; accountId: string; alias: string }; moduleSetId: string; defaultVarSetIds: string[]; }
export interface ModuleSet { moduleSetId: string; entries: { alias: string; source: string; version: string }[]; artifactS3?: string; }
export interface VarSet { varSetId: string; name: string; env: string; scope: "project"|"workspace"; vars: Record<string, unknown>; secretRefs: { provider: "ssm"|"secretsmanager"; path?: string; arn?: string; envName: string; }[]; }
export type RunStatus = "QUEUED"|"INIT"|"PLAN"|"APPLY"|"FAILED"|"SUCCEEDED"|"CANCELED";
export interface Run { runId: string; workspaceId: string; action: "plan"|"apply"; status: RunStatus; startedAt?: string; endedAt?: string; planS3?: string; applyLogS3?: string; }
TS

mkfile src/lib/utils/debounce.ts <<'TS'
export const debounce = <F extends (...args:any[])=>void>(fn:F, ms=300) => {
  let t: any; return (...args:any[]) => { clearTimeout(t); t = setTimeout(() => fn(...args), ms); };
};
TS

echo "==> Graph rules & cycles..."
mkfile src/lib/graph/rules.ts <<'TS'
export const allowed: Record<string, Record<string, string[]>> = {
  Lambda: { "reads-from": ["DynamoDB"], "invokes": ["Lambda","APIGateway"] },
  APIGateway: { "invokes": ["Lambda"] },
};
export const isAllowed = (fromType: string, relation: string, toType: string) =>
  !!allowed[fromType]?.[relation]?.includes(toType);
TS

mkfile src/lib/graph/cycles.ts <<'TS'
export function wouldCreateCycle(edges:{from:string;to:string}[], candidate:{from:string;to:string}) {
  const adj = new Map<string,string[]>(); for (const e of edges) adj.set(e.from,[...(adj.get(e.from)||[]),e.to]);
  const seen = new Set<string>(); const stack = [candidate.to];
  while (stack.length) { const n = stack.pop()!; if (n === candidate.from) return true;
    if (seen.has(n)) continue; seen.add(n); (adj.get(n)||[]).forEach(v=>stack.push(v)); }
  return false;
}
TS

echo "==> Editor state..."
mkfile src/lib/state/editorStore.ts <<'TS'
import { create } from "zustand";
import type { CanvasBundle, Node, Edge } from "@/lib/types";
type EditorState = {
  bundle?: CanvasBundle; selectedIds: string[]; dirty: boolean;
  setBundle: (b: CanvasBundle)=>void; upsertNode: (n: Node)=>void; upsertEdge: (e: Edge)=>void; removeById: (id: string)=>void; markClean: ()=>void;
};
export const useEditor = create<EditorState>()((set,get)=>({
  selectedIds: [], dirty:false,
  setBundle:(b)=>set({bundle:b,dirty:false}),
  upsertNode:(n)=>set(s=>({bundle: s.bundle?{...s.bundle,nodes: upsert(s.bundle.nodes,n)}:s.bundle, dirty:true})),
  upsertEdge:(e)=>set(s=>({bundle: s.bundle?{...s.bundle,edges: upsert(s.bundle.edges,e)}:s.bundle, dirty:true})),
  removeById:(id)=>set(s=>s.bundle?{bundle:{...s.bundle,nodes:s.bundle.nodes.filter(n=>n.nodeId!==id),edges:s.bundle.edges.filter(x=>x.edgeId!==id&&x.fromNodeId!==id&&x.toNodeId!==id)},dirty:true}:s),
  markClean:()=>set({dirty:false})
}));
const upsert = <T extends { [k:string]:any }>(arr:T[], item:T) => {
  const key = (item as any).nodeId ?? (item as any).edgeId;
  const idx = arr.findIndex(x => ((x as any).nodeId ?? (x as any).edgeId) === key);
  return idx>=0 ? arr.map((x,i)=>i===idx?item:x) : [...arr,item];
};
TS

echo "==> Minimal components..."
mkfile src/components/shell/AppShell.tsx <<'TS'
"use client";
import React from "react";
export default function AppShell({ children }:{children:React.ReactNode}) {
  return <div className="min-h-dvh flex flex-col">{children}</div>;
}
TS

mkfile src/components/common/Guard.tsx <<'TS'
"use client";
import React from "react";
import { useAuth } from "@/providers/AuthProvider";
export default function Guard({ children }:{children:React.ReactNode}) {
  const { loading } = useAuth(); if (loading) return <div>Loading…</div>;
  return <>{children}</>;
}
TS

mkfile src/components/common/PresenceBar.tsx <<'TS'
"use client";
export default function PresenceBar(){ return <div className="h-8 border-b px-3 text-sm flex items-center">Presence</div>; }
TS

mkfile src/components/diagram/DiagramPage.tsx <<'TS'
"use client";
import PresenceBar from "@/components/common/PresenceBar";
export default function DiagramPage(){ return (<div><PresenceBar/><div className="p-4">Diagram Editor Placeholder</div></div>); }
TS

# Create 5 placeholder diagram files (no heredoc to multiple files)
for f in Canvas.tsx Palette.tsx Inspector.tsx Console.tsx Toolbar.tsx; do
  mkfile "src/components/diagram/$f" <<'TS'
export default function Placeholder(){ return null; }
TS
done

mkfile src/components/runs/RunDrawer.tsx <<'TS'
export default function RunDrawer(){ return null; }
TS

echo "==> Next.js app routes..."
mkfile src/app/layout.tsx <<'TS'
import "./globals.css";
import { QueryProvider } from "@/providers/QueryProvider";
import { AuthProvider } from "@/providers/AuthProvider";
export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (<html lang="en"><body><AuthProvider><QueryProvider>{children}</QueryProvider></AuthProvider></body></html>);
}
export const metadata = { title: "Clutter" };
TS

mkfile src/app/page.tsx <<'TS'
export default function Home(){ return <main className="p-8">Clutter Landing</main>; }
TS

mkfile src/app/projects/page.tsx <<'TS'
import Guard from "@/components/common/Guard";
export default function Projects(){ return <Guard><main className="p-8">Projects</main></Guard>; }
TS

mkfile "src/app/projects/[pid]/page.tsx" <<'TS'
export default function ProjectOverview(){ return <main className="p-8">Project Overview</main>; }
TS

mkfile "src/app/projects/[pid]/diagrams/[did]/page.tsx" <<'TS'
import DiagramPage from "@/components/diagram/DiagramPage";
export default function DiagramRoute(){ return <DiagramPage/>; }
TS

mkfile "src/app/projects/[pid]/workspaces/[wid]/page.tsx" <<'TS'
export default function Workspace(){ return <main className="p-8">Workspace</main>; }
TS

mkfile "src/app/projects/[pid]/runs/page.tsx" <<'TS'
export default function Runs(){ return <main className="p-8">Runs</main>; }
TS

mkfile "src/app/projects/[pid]/settings/page.tsx" <<'TS'
export default function Settings(){ return <main className="p-8">Settings</main>; }
TS

echo "==> Patching tsconfig.json alias..."
node -e 'const fs=require("fs");const p="tsconfig.json";const j=JSON.parse(fs.readFileSync(p,"utf8"));j.compilerOptions=j.compilerOptions||{};j.compilerOptions.baseUrl=j.compilerOptions.baseUrl||".";j.compilerOptions.paths=Object.assign({"@/*":["src/*"]}, j.compilerOptions.paths||{});fs.writeFileSync(p, JSON.stringify(j,null,2));console.log("Updated tsconfig paths alias to @/* -> src/*");'

echo "✅ Done. Restart your TS server (Cmd+Shift+P → TypeScript: Restart TS Server) and run: pnpm dev"
