"use client";

import ErrorState from "@/components/common/ErrorState";
import ThemeScope from "@/components/common/ThemeScope";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { isBlocksDocument } from "@/lib/features/diagram/blocksDocument";
import { useDiagram } from "@/lib/features/diagram/hooks";
import { useDiagramEditorStore } from "@/lib/features/diagram/uiStore";
import { useOrganizations } from "@/lib/features/organization/hooks";
import { usePreferencesStore } from "@/lib/features/preferences/uiStore";
import { useProjectOrganization } from "@/lib/features/projects/hooks";
import { useMe } from "@/lib/features/user/hooks";
import { ArrowLeft, Info, Monitor } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import DiagramLoading from "@/app/(diagram)/loading";
import BlocksEditor from "./blocks/BlocksEditor";
import DiagramEditor from "./DiagramEditor";

// The editor is chosen per diagram: a saved document marked engine "blocks"
// opens B's editor; every other diagram opens A's editor, unchanged. Opening
// either one makes it "continue where you left off" for its team.
export default function EditorSwitch({ projectId, diagramId }: { projectId: string; diagramId: string }) {
  const meQ = useMe();
  const token = meQ.data?.token ?? null;
  const diagramQ = useDiagram(token, projectId, diagramId);
  const orgsQ = useOrganizations(token);
  const orgIds = useMemo(() => orgsQ.data?.map((o) => o.id) ?? null, [orgsQ.data]);
  const orgId = useProjectOrganization(token, projectId, orgIds).data ?? null;
  const rememberDiagram = usePreferencesStore((s) => s.rememberDiagram);
  const diagramName = diagramQ.data?.name ?? null;

  useEffect(() => {
    if (orgId && diagramName !== null) rememberDiagram(orgId, { projectId, diagramId, name: diagramName });
  }, [orgId, projectId, diagramId, diagramName, rememberDiagram]);

  if (meQ.isLoading || (token && diagramQ.isPending)) return <DiagramLoading />;
  if (!token) {
    return (
      <Centered>
        <ErrorState
          title="Sign in to open this diagram"
          description="Your session has ended."
          action={
            <Button asChild>
              <Link href="/login">Sign in</Link>
            </Button>
          }
        />
      </Centered>
    );
  }
  if (diagramQ.isError || !diagramQ.data) {
    return (
      <Centered>
        <ErrorState
          title="This diagram could not be opened"
          description="It may have been deleted, or it belongs to a team you are not in."
          onRetry={() => void diagramQ.refetch()}
          retrying={diagramQ.isFetching}
          action={
            <Button asChild>
              <Link href={`/projects/${projectId}/diagrams`}>Back to the project</Link>
            </Button>
          }
        />
      </Centered>
    );
  }
  if (isBlocksDocument(diagramQ.data.data)) {
    // B's editor has its own read-only view below 1024px.
    return <BlocksEditor key={diagramId} projectId={projectId} diagram={diagramQ.data} />;
  }
  // Below lg A's editor has no room, so the page says so instead. A's editor
  // stays mounted (only hidden), so a window made narrower keeps its unsaved work.
  return (
    <>
      <NarrowScreen projectId={projectId} />
      <div className="hidden h-dvh w-screen lg:block">
        {/* A's editor stays dark whatever the theme (ADR-006), on the background it was built on. */}
        <ThemeScope
          theme="a-editor"
          data-older-editor=""
          className="flex h-dvh w-screen flex-col bg-[radial-gradient(1200px_800px_at_30%_20%,rgba(40,60,90,0.55),transparent_60%),radial-gradient(900px_700px_at_85%_80%,rgba(0,140,120,0.35),transparent_55%),linear-gradient(180deg,#05070d_0%,#061021_45%,#061b1b_100%)] text-white">
          {/* Workaround: A's TopNav has its own Back, and A's files stay unchanged
              (ADR-006), so the wrapper hides it; the banner holds the one Back. */}
          <style>{`[data-older-editor] .react-flow__panel.top.left > div > div:last-child > button:first-child { display: none; }`}</style>
          <OlderEditorBar projectId={projectId} diagramId={diagramId} />
          {/* A's editor sizes itself to the screen; this box gives it the height under the bar. */}
          <div data-a-editor-body="" className="relative min-h-0 flex-1 [&>div]:!h-full">
            <DiagramEditor projectId={projectId} diagramId={diagramId} />
          </div>
        </ThemeScope>
      </div>
    </>
  );
}

// A bar above A's editor, which stays unchanged (ADR-006): the one way back,
// what this editor is, and what Deploy and Destroy need. It
// sits above A's "AWS account required" overlay (z-[100]), so a user without
// AWS still sees the way back.
function OlderEditorBar({ projectId, diagramId }: { projectId: string; diagramId: string }) {
  const router = useRouter();
  const dirty = useDiagramEditorStore((s) => !!s.state.byId[diagramId]?.dirty);
  const reset = useDiagramEditorStore((s) => s.actions.reset);
  const [asking, setAsking] = useState(false);
  const href = `/projects/${projectId}/diagrams`;
  const leave = () => {
    reset(diagramId);
    router.push(href);
  };

  return (
    <div
      role="note"
      data-testid="older-diagram-banner"
      className="bg-background text-muted-foreground relative z-[110] flex shrink-0 items-center gap-3 border-b px-3 py-1.5 text-xs">
      <Button type="button" variant="ghost" size="sm" onClick={() => (dirty ? setAsking(true) : leave())} className="text-foreground">
        <ArrowLeft aria-hidden />
        Back to the project
      </Button>
      <Info className="text-muted-foreground size-3.5 shrink-0" aria-hidden />
      <p className="min-w-0 flex-1 truncate">
        {/* Only what A's editor enforces: its AWS account overlay, and Deploy
            and Destroy off while there are changes to save (TopNav). */}
        <span className="text-foreground font-medium">Older AWS editor.</span> This diagram deploys to AWS only. Deploy and Destroy need a linked AWS
        account and a saved diagram.
      </p>
      <AlertDialog open={asking} onOpenChange={setAsking}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Leave without saving?</AlertDialogTitle>
            <AlertDialogDescription>Your changes to this diagram are not saved. If you leave now, they are lost.</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Stay</AlertDialogCancel>
            <AlertDialogAction onClick={leave}>Leave without saving</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

function NarrowScreen({ projectId }: { projectId: string }) {
  return (
    <div data-testid="narrow-screen" className="bg-background flex min-h-dvh items-center justify-center px-4 lg:hidden">
      <div className="surface-card w-full max-w-sm p-6 text-center">
        <Monitor className="text-link mx-auto size-8" aria-hidden />
        <h1 className="mt-4 text-lg font-semibold">This older diagram needs a wider screen</h1>
        <p className="text-muted-foreground mt-2 text-sm">Open it on a computer, or turn a tablet sideways. The window must be at least 1024 pixels wide.</p>
        <Button asChild className="mt-6">
          <Link href={`/projects/${projectId}/diagrams`}>Back to the project</Link>
        </Button>
      </div>
    </div>
  );
}

function Centered({ children }: { children: React.ReactNode }) {
  return (
    <div className="bg-background grid min-h-dvh place-items-center px-4">
      <div className="w-full max-w-md">{children}</div>
    </div>
  );
}
