"use client";

import CloudMark from "@/components/clouds/CloudMark";
import { useDialogFocus } from "@/components/common/useDialogFocus";
import { cloudLookFor } from "@/lib/features/catalog/display";
import { useCatalog } from "@/lib/features/catalog/hooks";
import type { CatalogEntry } from "@/lib/features/catalog/types";
import { fromBlocksDocument, toBlocksDocument } from "@/lib/features/diagram/blocksDocument";
import { BLANK_DIAGRAM_NAME, uniqueDiagramName } from "@/lib/features/diagram/diagramName";
import { useCreateDiagram } from "@/lib/features/diagram/hooks";
import { useExamples } from "@/lib/features/examples/hooks";
import { LEVEL_LABELS, type Example, type ExampleLevel } from "@/lib/features/examples/types";
import { Loader2, Plus, X } from "lucide-react";
import { useRouter } from "next/navigation";
import { useId, useRef, useState } from "react";
import { useProjectContext } from "../_contexts/ProjectContext";

const LEVEL_STYLES: Record<ExampleLevel, string> = {
  starter: "border-emerald-500/30 bg-emerald-500/10 text-emerald-300",
  intermediate: "border-sky-500/30 bg-sky-500/10 text-sky-300",
  advanced: "border-violet-500/30 bg-violet-500/10 text-violet-300",
};

const CARD =
  "group flex h-full w-full flex-col rounded-xl border p-4 text-left transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-teal-400 disabled:cursor-not-allowed disabled:opacity-50";

// The clouds an example fits: each alone, or, for a split example, all of
// them together in one deployment.
function Clouds({ example, catalog }: { example: Example; catalog: CatalogEntry[] }) {
  const together = (example.placements?.length ?? 0) > 0;
  const looks = example.clouds.map((c) => cloudLookFor(catalog, c));
  return (
    <span className="mt-auto flex flex-wrap items-center gap-1.5 pt-3">
      <span className="sr-only">{together ? `Runs on ${looks.map((l) => l.title).join(" and ")} together` : `Runs on ${looks.map((l) => l.title).join(" or ")}`}</span>
      {looks.map((l, i) => (
        <span key={l.connector} className="flex items-center gap-1.5" aria-hidden>
          {together && i > 0 && <span className="text-xs text-slate-500">+</span>}
          <span className="inline-flex items-center gap-1 rounded-md border border-slate-700 bg-slate-800/60 px-1.5 py-0.5 text-[11px] text-slate-300">
            <CloudMark cloud={l} size={14} />
            {l.shortName}
          </span>
        </span>
      ))}
    </span>
  );
}

// The new-diagram chooser: a blank canvas or one of B's built-in examples.
// Choosing one creates the diagram, saved with the example's blocks and
// links, and opens it.
export default function NewDiagramDialog({ onClose }: { onClose: () => void }) {
  const router = useRouter();
  const { project, projectId, token } = useProjectContext();
  const examplesQ = useExamples(token);
  const catalogQ = useCatalog(token);
  const createM = useCreateDiagram(token);
  const [choice, setChoice] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const titleId = useId();
  const panelRef = useRef<HTMLDivElement>(null);
  const busy = createM.isPending;
  const close = () => {
    if (!busy) onClose();
  };
  useDialogFocus(panelRef, close);

  const create = async (key: string, name: string, example: Example | null) => {
    if (!token || !project || busy) return;
    setChoice(key);
    setError(null);
    try {
      const doc = example ? fromBlocksDocument(example.document) : null;
      const created = await createM.mutateAsync({
        organizationId: project.organizationId,
        projectId,
        name: uniqueDiagramName(name, (project.diagrams ?? []).map((d) => d.name)),
        document: doc ? toBlocksDocument(doc.nodes, doc.edges) : undefined,
      });
      router.push(`/projects/${projectId}/diagram/${created.id}`);
    } catch {
      setError("Clutter could not create the diagram. Try again.");
      setChoice(null);
    }
  };

  const examples = examplesQ.data ?? [];
  const catalog = catalogQ.data ?? [];
  const spinner = (key: string) => busy && choice === key && <Loader2 className="h-4 w-4 animate-spin text-teal-300" aria-hidden />;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm" onClick={close}>
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
        data-testid="new-diagram-dialog"
        className="flex max-h-[90vh] w-full max-w-3xl flex-col rounded-2xl border border-slate-700 bg-slate-900 shadow-2xl outline-none"
        onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between border-b border-slate-800 px-6 py-4">
          <h2 id={titleId} className="text-lg font-semibold text-white">
            New diagram
          </h2>
          <button type="button" onClick={close} disabled={busy} aria-label="Close" className="rounded-md p-1.5 text-slate-400 hover:bg-slate-800 hover:text-white disabled:opacity-40">
            <X className="h-5 w-5" aria-hidden />
          </button>
        </div>

        <div className="overflow-y-auto p-6">
          <ul className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <li>
              <button
                type="button"
                data-testid="new-diagram-blank"
                disabled={busy}
                onClick={() => void create("blank", BLANK_DIAGRAM_NAME, null)}
                className={`${CARD} items-start border-dashed border-slate-600 bg-slate-900 hover:border-teal-500/60 hover:bg-slate-800/40`}>
                <span className="flex w-full items-center justify-between">
                  <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-slate-800 text-teal-300 group-hover:bg-teal-500/15">
                    <Plus className="h-5 w-5" aria-hidden />
                  </span>
                  {spinner("blank")}
                </span>
                <span className="mt-3 font-semibold text-white">Blank</span>
                <span className="mt-1 text-sm text-slate-400">An empty canvas.</span>
              </button>
            </li>

            {examplesQ.isLoading &&
              [0, 1, 2].map((i) => (
                <li key={i} aria-hidden className="h-[132px] animate-pulse rounded-xl border border-slate-800 bg-slate-800/40" />
              ))}

            {examples.map((ex) => (
              <li key={ex.id}>
                <button
                  type="button"
                  data-testid={`new-diagram-example-${ex.id}`}
                  disabled={busy}
                  onClick={() => void create(ex.id, ex.name, ex)}
                  className={`${CARD} border-slate-700 bg-slate-800/30 hover:border-teal-500/60 hover:bg-slate-800/60`}>
                  <span className="flex w-full items-center justify-between gap-2">
                    <span className="font-semibold text-white">{ex.name}</span>
                    {spinner(ex.id) || (
                      <span className={`shrink-0 rounded-full border px-2 py-0.5 text-[11px] font-medium ${LEVEL_STYLES[ex.level] ?? LEVEL_STYLES.starter}`}>
                        {LEVEL_LABELS[ex.level] ?? ex.level}
                      </span>
                    )}
                  </span>
                  <span className="mt-1 text-sm text-slate-400">{ex.description}</span>
                  <Clouds example={ex} catalog={catalog} />
                </button>
              </li>
            ))}
          </ul>

          {examplesQ.isLoading && (
            <p role="status" className="sr-only">
              Loading examples
            </p>
          )}
          {examplesQ.isError && (
            <p role="alert" className="mt-4 text-sm text-amber-300">
              Examples could not load. You can still start blank.
            </p>
          )}
          {examplesQ.isSuccess && examples.length === 0 && <p className="mt-4 text-sm text-slate-400">No examples are available yet.</p>}
          {error && (
            <p role="alert" className="mt-4 text-sm text-red-400">
              {error}
            </p>
          )}
        </div>
      </div>
    </div>
  );
}
