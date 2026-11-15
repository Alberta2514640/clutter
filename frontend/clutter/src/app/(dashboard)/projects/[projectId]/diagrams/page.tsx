"use client";

import ConfirmDialog from "@/components/common/ConfirmDialog";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { isBlocksDocument } from "@/lib/features/diagram/blocksDocument";
import { editedLabel } from "@/lib/features/diagram/edited";
import { useCreateDiagram, useDeleteDiagram, useDiagrams } from "@/lib/features/diagram/hooks";
import { isConflict } from "@/lib/features/user/session";
import { FileText, Loader2, Trash2 } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import { useProjectContext } from "../_contexts/ProjectContext";

type DeleteTarget = { id: string; name: string } | null;

export default function ProjectDiagramsPage() {
  const router = useRouter();
  const { projectId, token, project } = useProjectContext();

  const createDiagramM = useCreateDiagram(token);
  const [createError, setCreateError] = useState<string | null>(null);

  // The project lists only ids and names; the diagrams themselves say which
  // editor each one uses, for the "AWS only" mark.
  const diagramsQ = useDiagrams(token, projectId);
  const loadedById = useMemo(() => new Map((diagramsQ.data ?? []).map((d) => [d.id, d])), [diagramsQ.data]);
  // Read once per render; the cards' "Edited 5 minutes ago" needs no ticking.
  const [now] = useState(() => new Date());

  const deleteDiagramM = useDeleteDiagram(token);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<DeleteTarget>(null);
  const [showDeleted, setShowDeleted] = useState(false);

  const handleCreateFromScratch = async () => {
    if (!token || !projectId || !project) return;
    setCreateError(null);
    try {
      const created = await createDiagramM.mutateAsync({
        organizationId: project.organizationId,
        projectId,
        name: "New Diagram",
      });
      router.push(`/projects/${projectId}/diagram/${created.id}`);
    } catch {
      setCreateError("Clutter could not create the diagram. Try again.");
    }
  };

  const handleDeleteConfirm = async () => {
    if (!deleteTarget || !project) return;
    setDeleteError(null);
    try {
      await deleteDiagramM.mutateAsync({
        organizationId: project.organizationId,
        projectId,
        diagramId: deleteTarget.id,
      });
    } catch (err) {
      setDeleteError(
        isConflict(err)
          ? "This diagram still has live deployments. Destroy them on the diagram's Deploy tab, then delete it."
          : "Clutter could not delete the diagram. Try again.",
      );
      return;
    }
    setDeleteTarget(null);
    setShowDeleted(true);
    setTimeout(() => setShowDeleted(false), 2000);
  };

  const diagrams = project?.diagrams ?? [];
  const hasDiagrams = diagrams.length > 0;

  if (!project) {
    return (
      <div role="status" className="py-4 text-center text-gray-400">
        Loading the project...
      </div>
    );
  }

  return (
    // The main area's padding lines the grid up with the project header.
    <div>
      {deleteTarget && (
        <ConfirmDialog
          title="Delete diagram"
          confirmLabel="Delete"
          pendingLabel="Deleting…"
          isPending={deleteDiagramM.isPending}
          error={deleteError}
          onCancel={() => setDeleteTarget(null)}
          onConfirm={() => void handleDeleteConfirm()}>
          <p>
            Delete <span className="text-white font-medium">&quot;{deleteTarget.name}&quot;</span>? This cannot be undone.
          </p>
          <p>If the diagram has live deployments, Clutter refuses until you destroy them on its Deploy tab.</p>
        </ConfirmDialog>
      )}

      {/* ── Delete toast ── */}
      {showDeleted && (
        <div role="status" className="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 rounded-lg bg-slate-900 border border-slate-700 px-4 py-2 text-sm text-white shadow-lg flex items-center gap-2">
          <Trash2 className="h-3.5 w-3.5 text-slate-400" aria-hidden="true" />
          Diagram deleted
        </div>
      )}

      {/* ── Error banner ── */}
      {createError && (
        <div role="alert" className="mb-6 p-4 rounded-lg bg-red-900/20 border border-red-800">
          <p className="text-sm text-red-400">{createError}</p>
        </div>
      )}

      {/* ── Empty state ── */}
      {!hasDiagrams && (
        <div className="text-center mb-12">
          <h2 className="text-3xl font-bold text-white mb-2">No diagrams yet</h2>
          <p className="text-gray-400 text-lg">Draw your first diagram in building blocks, then deploy it to your connected clouds.</p>
          <div className="mx-auto mt-12 max-w-md">
            <Card className="bg-slate-800/50 border-slate-700/50 hover:border-teal-500/50 transition-all">
              <CardContent className="p-8 text-center">
                <div className="inline-flex items-center justify-center w-16 h-16 mb-6 rounded-xl bg-slate-700/50">
                  <FileText className="w-8 h-8 text-gray-400" aria-hidden="true" />
                </div>
                <h3 className="text-lg font-semibold text-white mb-2">Start from scratch</h3>
                <p className="text-sm text-gray-400 mb-6">An empty canvas with the building blocks your clouds offer.</p>
                <Button onClick={() => void handleCreateFromScratch()} disabled={createDiagramM.isPending} className="bg-teal-600 hover:bg-teal-500 text-white">
                  {createDiagramM.isPending ? (
                    <>
                      <Loader2 className="w-4 h-4 mr-2 animate-spin" aria-hidden="true" />
                      Creating...
                    </>
                  ) : (
                    "New diagram"
                  )}
                </Button>
              </CardContent>
            </Card>
          </div>
        </div>
      )}

      {/* ── Diagrams grid ── */}
      {hasDiagrams && (
        <ul className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {diagrams.map((diagram) => {
            const isDeleting = deleteDiagramM.isPending && deleteDiagramM.variables?.diagramId === diagram.id;
            const loaded = loadedById.get(diagram.id);
            // Unknown until the diagrams load; only a loaded, unmarked diagram is A's.
            const isOlderAwsDiagram = !!loaded && !isBlocksDocument(loaded.data);
            const edited = editedLabel(loaded?.latestUpdateAt ?? loaded?.createdAt, now);

            return (
              <li key={diagram.id}>
                <Card className={`relative h-full bg-slate-900/40 border-slate-800 hover:border-teal-500/40 focus-within:border-teal-500/60 transition group ${isDeleting ? "opacity-50" : ""}`}>
                  <CardContent className="p-5">
                    <div className="flex items-start justify-between gap-4 mb-3">
                      <div className="w-12 h-12 flex items-center justify-center rounded-lg bg-slate-800 group-hover:bg-teal-500/20 transition-colors">
                        <FileText className="w-6 h-6 text-gray-400 group-hover:text-teal-400 transition-colors" aria-hidden="true" />
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          setDeleteError(null);
                          setDeleteTarget({ id: diagram.id, name: diagram.name });
                        }}
                        disabled={isDeleting}
                        aria-label={`Delete diagram ${diagram.name}`}
                        className="relative z-10 h-8 w-8 p-0 hover:bg-red-900/30 hover:text-red-400 text-gray-400">
                        <Trash2 className="w-4 h-4" aria-hidden="true" />
                      </Button>
                    </div>
                    <div className="min-w-0 space-y-2">
                      {/* The title link covers the card, so the whole card opens the diagram. */}
                      <Link
                        href={`/projects/${projectId}/diagram/${diagram.id}`}
                        className="block truncate text-white font-semibold outline-none after:absolute after:inset-0 after:rounded-xl focus-visible:underline">
                        {diagram.name}
                      </Link>
                      {edited && <p className="text-xs text-gray-400">{edited}</p>}
                      {isOlderAwsDiagram && (
                        <span className="inline-block rounded-full border border-amber-500/40 bg-amber-500/10 px-2 py-0.5 text-xs text-amber-200">AWS only, older editor</span>
                      )}
                    </div>
                    {isDeleting && (
                      <div className="absolute inset-0 flex items-center justify-center bg-slate-900/60 rounded-lg">
                        <p className="text-sm text-gray-300">Deleting...</p>
                      </div>
                    )}
                  </CardContent>
                </Card>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
