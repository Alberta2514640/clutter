"use client";

import CloudLogo, { CLOUD_LOGOS } from "@/components/common/CloudLogo";
import LevelBadge from "@/components/common/LevelBadge";
import { Button } from "@/components/ui/button";
import { RadioCard, RadioCardDescription, RadioCardGroup, RadioCardTitle } from "@/components/ui/radio-card";
import { Skeleton } from "@/components/ui/skeleton";
import { useExamples } from "@/lib/features/examples/hooks";
import type { Example } from "@/lib/features/examples/types";
import { PROJECTS_PATH } from "@/lib/features/organization/home";
import { usePreferencesStore } from "@/lib/features/preferences/uiStore";
import { orderExamplesForCloud } from "@/lib/features/projects/exampleOrder";
import { useCreateProject, useProjects } from "@/lib/features/projects/hooks";
import { FIRST_PROJECT_NAME } from "@/lib/features/projects/rules";
import { useStartDiagram } from "@/lib/features/projects/startDiagram";
import { useIsClient } from "@/lib/features/user/useIsClient";
import { Plus } from "lucide-react";
import { useRouter } from "next/navigation";
import { useId, useMemo, useState } from "react";
import { StartArt } from "../_components/StepArt";
import StepFrame, { StepError, StepLoading } from "../_components/StepFrame";
import { CLOUD_PATH } from "../_components/steps";
import { useOnboardingSession } from "../_components/useOnboardingSession";

// Step 4 of 4: a starting point. Choosing one makes "My first project" (A's
// project create) and a building-block diagram from it (the same create as
// "New diagram"), then opens the editor. Examples that fit the cloud from
// step 3 come first.
const BLANK = "blank";

const cloudName = (connector: string) => CLOUD_LOGOS[connector]?.shortName ?? connector;

function ExampleClouds({ example }: { example: Example }) {
  const together = (example.placements?.length ?? 0) > 0;
  const names = example.clouds.map(cloudName);
  return (
    <span className="mt-2 flex flex-wrap items-center gap-1.5">
      <span className="sr-only">{together ? `Runs on ${names.join(" and ")} together.` : `Runs on ${names.join(" or ")}.`}</span>
      {example.clouds.map((c, i) => (
        <span key={c} aria-hidden="true" className="flex items-center gap-1.5">
          {together && i > 0 && <span className="text-muted-foreground text-xs">+</span>}
          <span className="border-border text-muted-foreground inline-flex items-center gap-1 rounded-full border py-0.5 pr-2 pl-1 text-[11px] font-medium">
            <CloudLogo connector={c} size={12} />
            {cloudName(c)}
          </span>
        </span>
      ))}
    </span>
  );
}

export default function StartStep() {
  const router = useRouter();
  const session = useOnboardingSession("after-team");
  const organizationId = session.organization?.id ?? null;
  const examplesQ = useExamples(session.token);
  const projectsQ = useProjects(session.token, organizationId);
  const createProject = useCreateProject(session.token);
  const { start, isPending: startingDiagram } = useStartDiagram(session.token);
  const isClient = useIsClient();
  const firstCloud = usePreferencesStore((s) => s.firstCloud);
  const [choice, setChoice] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const titleId = useId();

  const cloud = isClient ? firstCloud : null;
  const examples = useMemo(() => orderExamplesForCloud(examplesQ.data ?? [], cloud), [examplesQ.data, cloud]);

  if (session.isError) return <StepError onRetry={session.retry} retrying={session.retrying} />;
  if (!session.ready || !organizationId) return <StepLoading />;

  const pending = createProject.isPending || startingDiagram;

  const create = async () => {
    if (!choice || pending) return;
    setError(null);
    const example = choice === BLANK ? null : (examples.find((e) => e.id === choice) ?? null);
    try {
      // Coming back to this step reuses the project it made before.
      const existing = (projectsQ.data ?? []).find((p) => p.name === FIRST_PROJECT_NAME);
      const project = existing ?? (await createProject.mutateAsync({ organizationId, name: FIRST_PROJECT_NAME }));
      const diagram = await start({
        organizationId,
        projectId: project.id,
        takenNames: (project.diagrams ?? []).map((d) => d.name),
        example,
      });
      router.push(`/projects/${project.id}/diagram/${diagram.id}`);
    } catch {
      setError("Clutter could not create your first diagram. Try again.");
    }
  };

  return (
    <StepFrame
      title="Pick a starting point"
      titleId={titleId}
      illustration={<StartArt blank={choice === BLANK} document={examples.find((e) => e.id === choice)?.document} />}
      description={
        <>
          Clutter makes a project called &ldquo;{FIRST_PROJECT_NAME}&rdquo; with this diagram and opens it. Nothing is built until you deploy.
        </>
      }
      onContinue={() => void create()}
      continueLabel="Open the editor"
      continueDisabled={!choice}
      pending={pending}
      pendingLabel="Making your diagram…"
      backHref={CLOUD_PATH}
      skip={{ label: "Skip for now", onSkip: () => router.push(PROJECTS_PATH) }}
      error={error}>
      <RadioCardGroup aria-labelledby={titleId} value={choice ?? ""} onValueChange={setChoice} onEnter={() => void create()} className="sm:grid-cols-2" disabled={pending}>
        {examples.map((example) => (
          <RadioCard key={example.id} value={example.id} data-testid={`onboarding-example-${example.id}`} className="items-start">
            <span className="min-w-0">
              <span className="flex flex-wrap items-center gap-2">
                <RadioCardTitle>{example.name}</RadioCardTitle>
                <LevelBadge level={example.level} />
              </span>
              <RadioCardDescription>{example.description}</RadioCardDescription>
              <ExampleClouds example={example} />
            </span>
          </RadioCard>
        ))}
        {examplesQ.isLoading &&
          [0, 1, 2].map((i) => (
            <div key={i} role="presentation">
              <Skeleton className="h-[116px] w-full rounded-xl" />
            </div>
          ))}
        <RadioCard value={BLANK} data-testid="onboarding-example-blank" className="items-start border-dashed">
          <span className="flex min-w-0 gap-3">
            <span className="bg-muted text-link flex size-9 shrink-0 items-center justify-center rounded-lg">
              <Plus className="size-5" aria-hidden="true" />
            </span>
            <span className="min-w-0">
              <RadioCardTitle>Blank</RadioCardTitle>
              <RadioCardDescription>An empty canvas to draw on.</RadioCardDescription>
            </span>
          </span>
        </RadioCard>
      </RadioCardGroup>

      {examplesQ.isLoading && (
        <p role="status" className="sr-only">
          Loading examples
        </p>
      )}
      {examplesQ.isError && (
        <div role="alert" className="text-muted-foreground mt-4 flex flex-wrap items-center gap-2 text-sm">
          The examples could not load. You can still start blank.
          <Button type="button" variant="outline" size="xs" onClick={() => void examplesQ.refetch()}>
            Retry
          </Button>
        </div>
      )}
    </StepFrame>
  );
}
