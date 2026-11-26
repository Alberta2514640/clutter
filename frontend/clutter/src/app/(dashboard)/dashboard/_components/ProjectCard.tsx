import StatusDot from "@/components/common/StatusDot";
import { Skeleton } from "@/components/ui/skeleton";
import type { DeploySummary } from "@/lib/features/projects/deploymentSummary";
import type { Project } from "@/lib/features/projects/types";
import { plural } from "@/lib/text";
import { ChevronRight, Layers } from "lucide-react";
import Link from "next/link";

type Props = {
  project: Project;
  // Its deployments' summary; undefined while they load, null when they failed.
  deployments: DeploySummary | undefined | null;
  deploymentsLoading: boolean;
};

// One project on Projects: its name, its description when it has one, how many
// diagrams it holds and how many of their deployments are live or running.
export default function ProjectCard({ project, deployments, deploymentsLoading }: Props) {
  const diagrams = project.diagrams?.length ?? 0;
  const description = project.description?.trim();

  return (
    <article
      data-testid="project-card"
      data-spotlight
      className="group surface-card surface-card-interactive spotlight text-card-foreground relative flex h-full flex-col p-5">
      <div className="flex items-start justify-between gap-3">
        <h3 className="min-w-0 text-base font-semibold">
          {/* The title link covers the card, so the whole card opens the project. */}
          <Link
            href={`/projects/${project.id}/diagrams`}
            className="block truncate rounded-sm outline-none after:absolute after:inset-0 after:rounded-xl focus-visible:underline">
            {project.name}
          </Link>
        </h3>
        <ChevronRight
          className="text-muted-foreground group-hover:text-link size-4 shrink-0 transition-[color,translate] group-hover:translate-x-0.5"
          aria-hidden="true"
        />
      </div>
      {description && <p className="text-muted-foreground mt-1.5 line-clamp-2 text-sm">{description}</p>}

      <div className="text-muted-foreground mt-auto flex flex-wrap items-center gap-x-4 gap-y-2 pt-5 text-sm">
        <span className="inline-flex items-center gap-1.5">
          <Layers className="size-4" aria-hidden="true" />
          {plural(diagrams, "diagram", "diagrams")}
        </span>
        {deploymentsLoading ? (
          <Skeleton className="h-4 w-16" />
        ) : deployments ? (
          <>
            {deployments.running > 0 && <StatusDot tone="primary" pulse label={`${deployments.running} running`} />}
            {deployments.live > 0 && <StatusDot tone="success" label={`${deployments.live} live`} />}
          </>
        ) : null}
      </div>
    </article>
  );
}
