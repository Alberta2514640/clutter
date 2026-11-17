"use client";

import { isBlocksDocument } from "@/lib/features/diagram/blocksDocument";
import { useDiagram } from "@/lib/features/diagram/hooks";
import { useMe } from "@/lib/features/user/hooks";
import { Info, Monitor } from "lucide-react";
import Link from "next/link";
import BlocksEditor from "./blocks/BlocksEditor";
import DiagramEditor from "./DiagramEditor";

// The editor is chosen per diagram: a saved document marked engine "blocks"
// opens B's editor; every other diagram opens A's editor, unchanged.
export default function EditorSwitch({ projectId, diagramId }: { projectId: string; diagramId: string }) {
  const meQ = useMe();
  const token = meQ.data?.token ?? null;
  const diagramQ = useDiagram(token, projectId, diagramId);

  if (meQ.isLoading || (token && diagramQ.isPending)) {
    return (
      <Screen>
        <div role="status" className="rounded-lg bg-neutral-900 px-4 py-2 text-sm text-white">
          Loading diagram…
        </div>
      </Screen>
    );
  }
  if (!token) {
    return (
      <Screen>
        <p className="text-sm text-slate-300">
          Sign in to open this diagram. <Link href="/login" className="text-teal-300 hover:underline">Sign in</Link>
        </p>
      </Screen>
    );
  }
  if (diagramQ.isError || !diagramQ.data) {
    return (
      <Screen>
        <div role="alert" className="space-y-2 text-center text-sm text-slate-300">
          <p>This diagram could not be opened.</p>
          <Link href={`/projects/${projectId}/diagrams`} className="text-teal-300 hover:underline">
            Back to the project
          </Link>
        </div>
      </Screen>
    );
  }
  const isBlocks = isBlocksDocument(diagramQ.data.data);
  // Below lg the canvas has no room, so the page says so instead. The editor
  // stays mounted (only hidden), so a window made narrower keeps its unsaved work.
  return (
    <>
      <NarrowScreen projectId={projectId} />
      <div className="hidden h-screen w-screen lg:block">
        {isBlocks ? (
          <BlocksEditor key={diagramId} projectId={projectId} diagram={diagramQ.data} />
        ) : (
          <div className="flex h-screen w-screen flex-col">
            <OlderEditorBar projectId={projectId} />
            {/* A's editor sizes itself to the screen; this box gives it the height under the bar. */}
            <div className="relative min-h-0 flex-1 [&>div]:!h-full">
              <DiagramEditor projectId={projectId} diagramId={diagramId} />
            </div>
          </div>
        )}
      </div>
    </>
  );
}

// A thin bar above A's editor, which stays unchanged (ADR-006). It sits above
// A's "AWS account required" overlay (z-[100]), so a user without AWS still
// sees the way back to the project.
function OlderEditorBar({ projectId }: { projectId: string }) {
  return (
    <div
      role="note"
      data-testid="older-diagram-banner"
      className="relative z-[110] flex shrink-0 items-center gap-2 border-b border-slate-800 bg-slate-950 px-4 py-1.5 text-xs text-slate-300">
      <Info className="h-3.5 w-3.5 shrink-0 text-teal-300" aria-hidden />
      <p className="min-w-0 flex-1 truncate">
        <span className="font-medium text-white">Older AWS editor.</span> This diagram deploys to AWS only. New diagrams use building blocks and can deploy to
        any cloud.
      </p>
      <Link href={`/projects/${projectId}/diagrams`} className="shrink-0 text-teal-300 hover:underline">
        Back to the project
      </Link>
    </div>
  );
}

function NarrowScreen({ projectId }: { projectId: string }) {
  return (
    <div data-testid="narrow-screen" className="flex min-h-screen items-center justify-center bg-slate-950 px-4 lg:hidden">
      <div className="w-full max-w-sm rounded-xl border border-slate-800 bg-slate-900/70 p-6 text-center">
        <Monitor className="mx-auto h-8 w-8 text-teal-300" aria-hidden />
        <h1 className="mt-4 text-lg font-semibold text-white">The editor needs a wider screen</h1>
        <p className="mt-2 text-sm text-slate-400">Open this diagram on a computer, or turn a tablet sideways. The window must be at least 1024 pixels wide.</p>
        <Link
          href={`/projects/${projectId}/diagrams`}
          className="mt-6 inline-flex items-center justify-center rounded-md bg-teal-600 px-4 py-2 text-sm font-medium text-white hover:bg-teal-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-teal-300">
          Back to the project
        </Link>
      </div>
    </div>
  );
}

function Screen({ children }: { children: React.ReactNode }) {
  return <div className="grid h-screen w-screen place-items-center bg-slate-950">{children}</div>;
}
