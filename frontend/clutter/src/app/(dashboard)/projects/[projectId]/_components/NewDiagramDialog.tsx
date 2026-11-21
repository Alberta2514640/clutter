"use client";

import CloudLogo from "@/components/common/CloudLogo";
import LevelBadge from "@/components/common/LevelBadge";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { cloudLookFor } from "@/lib/features/catalog/display";
import { useCatalog } from "@/lib/features/catalog/hooks";
import type { CatalogEntry } from "@/lib/features/catalog/types";
import { BLANK_DIAGRAM_NAME, DIAGRAM_NAME_MAX, diagramNameError } from "@/lib/features/diagram/diagramName";
import { useExamples } from "@/lib/features/examples/hooks";
import type { Example } from "@/lib/features/examples/types";
import { usePreferencesStore } from "@/lib/features/preferences/uiStore";
import { orderExamplesForCloud } from "@/lib/features/projects/exampleOrder";
import { useStartDiagram } from "@/lib/features/projects/startDiagram";
import { cn } from "@/lib/utils";
import { Loader2, Plus } from "lucide-react";
import { NEW_DIAGRAM_QUERY } from "@/lib/features/projects/checklist";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useEffect, useId, useState } from "react";
import { useProjectContext } from "../_contexts/ProjectContext";

const CARD =
  "group surface-card surface-card-interactive focus-visible:ring-ring/50 flex h-full w-full flex-col p-4 text-left outline-none focus-visible:ring-[3px] disabled:cursor-not-allowed disabled:opacity-50";

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
          {together && i > 0 && <span className="text-muted-foreground text-xs">+</span>}
          <span className="bg-muted text-muted-foreground inline-flex items-center gap-1 rounded-full border py-0.5 pr-2 pl-1 text-[11px]">
            <CloudLogo connector={l.connector} shortName={l.shortName} src={l.logoSrc} size={14} />
            {l.shortName}
          </span>
        </span>
      ))}
    </span>
  );
}

// "New diagram": the project's one button for it, which opens the chooser.
// The button is the dialog's trigger, so closing the dialog puts focus back
// on it. The chooser mounts only while open, so its name starts fresh. The
// one button with openFromLink (the project header's, always mounted) also
// opens when the address asks (?new=diagram, from the Getting started
// checklist), then drops the query so a reload does not ask again.
export default function NewDiagramButton({ className, openFromLink = false }: { className?: string; openFromLink?: boolean }) {
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const asked = openFromLink && `new=${params.get("new")}` === NEW_DIAGRAM_QUERY;
  const [seenAsk, setSeenAsk] = useState(false);
  if (asked !== seenAsk) {
    setSeenAsk(asked);
    if (asked) setOpen(true);
  }
  useEffect(() => {
    if (asked) router.replace(pathname, { scroll: false });
  }, [asked, pathname, router]);
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next && busy) return;
        setOpen(next);
      }}>
      <DialogTrigger asChild>
        <Button type="button" className={className}>
          <Plus aria-hidden />
          New diagram
        </Button>
      </DialogTrigger>
      {open && <NewDiagramDialog onBusyChange={setBusy} />}
    </Dialog>
  );
}

// The new-diagram chooser: a name, then a blank canvas or one of B's
// built-in examples. A name left as it is takes the example's own name.
// Choosing one creates the diagram, saved with the example's blocks and
// links, and opens it.
function NewDiagramDialog({ onBusyChange }: { onBusyChange: (busy: boolean) => void }) {
  const router = useRouter();
  const { project, projectId, token } = useProjectContext();
  const examplesQ = useExamples(token);
  const catalogQ = useCatalog(token);
  const firstCloud = usePreferencesStore((s) => s.firstCloud);
  const { start, isPending: busy } = useStartDiagram(token);
  const [choice, setChoice] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [name, setName] = useState(BLANK_DIAGRAM_NAME);
  const [named, setNamed] = useState(false);
  const nameId = useId();
  const nameProblem = named ? diagramNameError(name) : null;

  const create = async (key: string, example: Example | null) => {
    if (!token || !project || busy || nameProblem) return;
    setChoice(key);
    setError(null);
    try {
      const created = await start({
        organizationId: project.organizationId,
        projectId,
        takenNames: (project.diagrams ?? []).map((d) => d.name),
        example,
        // An untouched name takes the example's own name.
        name: named ? name : undefined,
      });
      router.push(`/projects/${projectId}/diagram/${created.id}`);
    } catch {
      setError("Clutter could not create the diagram. Try again.");
      setChoice(null);
    }
  };

  // Examples that fit the cloud chosen at onboarding come first.
  const examples = orderExamplesForCloud(examplesQ.data ?? [], firstCloud);
  const catalog = catalogQ.data ?? [];
  const spinner = (key: string) => busy && choice === key && <Loader2 className="text-link size-4 animate-spin" aria-hidden />;

  // While a diagram is being made, the dialog stays open (NewDiagramButton).
  useEffect(() => onBusyChange(busy), [busy, onBusyChange]);

  return (
    <>
      <DialogContent data-testid="new-diagram-dialog" className="flex max-h-[90dvh] max-w-3xl flex-col gap-0 p-0 sm:max-w-3xl">
        <DialogHeader className="border-b px-6 pt-5 pb-4">
          <DialogTitle>New diagram</DialogTitle>
          <DialogDescription>Name it, then start blank or from an example.</DialogDescription>
        </DialogHeader>

        <div className="space-y-5 overflow-y-auto p-6">
          <div className="space-y-1.5">
            <label htmlFor={nameId} className="block text-sm font-medium">
              Name
            </label>
            <Input
              id={nameId}
              value={name}
              maxLength={DIAGRAM_NAME_MAX}
              onChange={(e) => {
                setName(e.target.value);
                setNamed(true);
              }}
              onFocus={(e) => {
                if (!named) e.currentTarget.select();
              }}
              aria-invalid={!!nameProblem || undefined}
              aria-describedby={`${nameId}-hint`}
              className="max-w-sm"
            />
            <p id={`${nameId}-hint`} className={cn("text-xs", nameProblem ? "text-danger" : "text-muted-foreground")}>
              {nameProblem ?? "Leave it as it is and an example keeps its own name."}
            </p>
          </div>

          <ul className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <li>
              <button
                type="button"
                data-testid="new-diagram-blank"
                disabled={busy || !!nameProblem}
                onClick={() => void create("blank", null)}
                className={cn(CARD, "border-dashed")}>
                <span className="flex w-full items-center justify-between">
                  <span className="bg-primary/10 text-link flex size-9 items-center justify-center rounded-lg">
                    <Plus className="size-5" aria-hidden />
                  </span>
                  {spinner("blank")}
                </span>
                <span className="mt-3 font-semibold">Blank</span>
                <span className="text-muted-foreground mt-1 text-sm">An empty canvas.</span>
              </button>
            </li>

            {examplesQ.isLoading &&
              [0, 1, 2].map((i) => (
                <li key={i} aria-hidden>
                  <Skeleton className="h-[132px] rounded-xl" />
                </li>
              ))}

            {examples.map((ex) => (
              <li key={ex.id}>
                <button
                  type="button"
                  data-testid={`new-diagram-example-${ex.id}`}
                  disabled={busy || !!nameProblem}
                  onClick={() => void create(ex.id, ex)}
                  className={CARD}>
                  <span className="flex w-full items-center justify-between gap-2">
                    <span className="font-semibold">{ex.name}</span>
                    {spinner(ex.id) || <LevelBadge level={ex.level} />}
                  </span>
                  <span className="text-muted-foreground mt-1 text-sm">{ex.description}</span>
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
            <p role="alert" className="text-warning text-sm">
              Examples could not load. You can still start blank.
            </p>
          )}
          {examplesQ.isSuccess && examples.length === 0 && <p className="text-muted-foreground text-sm">No examples are available yet.</p>}
          {error && (
            <p role="alert" className="text-danger text-sm">
              {error}
            </p>
          )}
        </div>
      </DialogContent>
    </>
  );
}
