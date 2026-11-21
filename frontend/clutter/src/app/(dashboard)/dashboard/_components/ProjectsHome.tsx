"use client";

import IsoStack from "@/components/brand/IsoStack";
import EmptyState from "@/components/common/EmptyState";
import FirstDeployChecklist from "@/components/common/FirstDeployChecklist";
import ErrorState from "@/components/common/ErrorState";
import PageHeader from "@/components/common/PageHeader";
import SpotlightGroup from "@/components/common/SpotlightGroup";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useOrgDeployments } from "@/lib/features/deployments/orgHooks";
import type { Organization } from "@/lib/features/organization/types";
import { summarizeByProject } from "@/lib/features/projects/deploymentSummary";
import { useProjects } from "@/lib/features/projects/hooks";
import { Plus } from "lucide-react";
import Link from "next/link";
import { useMemo } from "react";
import ConnectionsSummary from "./ConnectionsSummary";
import ContinueCard from "./ContinueCard";
import OlderRuns from "./OlderRuns";
import ProjectCard from "./ProjectCard";

const NEW_PROJECT_HREF = "/projects/new";

const newProjectButton = (
  <Button asChild>
    <Link href={NEW_PROJECT_HREF}>
      <Plus aria-hidden="true" />
      New project
    </Link>
  </Button>
);

// Projects: where the team's work is, what it has live, and the clouds it can
// build in. A new team sees how to get to a first deploy.
export default function ProjectsHome({ token, organization }: { token: string; organization: Organization }) {
  const organizationId = organization.id;
  const projectsQ = useProjects(token, organizationId);
  const deploymentsQ = useOrgDeployments(token, organizationId);

  const projects = useMemo(
    () => [...(projectsQ.data ?? [])].sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: "base" })),
    [projectsQ.data],
  );
  const byProject = useMemo(() => summarizeByProject(deploymentsQ.data ?? []), [deploymentsQ.data]);
  const noProjects = projectsQ.isSuccess && projects.length === 0;

  return (
    <>
      <PageHeader
        eyebrow={organization.name}
        title="Projects"
        description="What this team is building. A project holds your diagrams."
        actions={noProjects ? undefined : newProjectButton}
      />

      {/* Getting started: the same checklist as the editor's, until the team's first live deployment. */}
      <div className="empty:hidden mb-8">
        <FirstDeployChecklist organizationId={organizationId} variant="card" />
      </div>

      <ContinueCard organizationId={organizationId} projects={projects} />

      <div className="grid gap-8 lg:grid-cols-[minmax(0,1fr)_18rem]">
        <section aria-labelledby="projects-heading" className="min-w-0">
          <h2 id="projects-heading" className="micro-label mb-3">
            Your projects
          </h2>

          {projectsQ.isLoading ? (
            <div role="status" className="grid gap-4 sm:grid-cols-2">
              <span className="sr-only">Loading projects…</span>
              {[0, 1, 2, 3].map((i) => (
                <Skeleton key={i} className="h-36 rounded-xl" />
              ))}
            </div>
          ) : projectsQ.isError ? (
            <ErrorState
              title="Projects could not load"
              description="Check your connection, then try again."
              onRetry={() => void projectsQ.refetch()}
              retrying={projectsQ.isFetching}
            />
          ) : noProjects ? (
            <EmptyState
              illustration={<IsoStack preset="cluster" accent="brand" size={28} assemble idle="lift" />}
              title="No projects yet"
              description="A project holds diagrams of what you want to build. Make one, then draw your setup in building blocks."
              action={newProjectButton}
            />
          ) : (
            <>
              <SpotlightGroup as="ul" className="grid gap-4 sm:grid-cols-2">
                {projects.map((project) => (
                  <li key={project.id} className="animate-fade-in">
                    <ProjectCard
                      project={project}
                      deploymentsLoading={deploymentsQ.isLoading}
                      deployments={deploymentsQ.isError ? null : byProject.get(project.id)}
                    />
                  </li>
                ))}
              </SpotlightGroup>
              {deploymentsQ.isError && (
                <ErrorState
                  variant="inline"
                  className="mt-4"
                  description="Deployment status could not load, so live counts are missing."
                  onRetry={() => void deploymentsQ.refetch()}
                  retrying={deploymentsQ.isFetching}
                />
              )}
            </>
          )}
        </section>

        <aside className="min-w-0 space-y-6">
          <ConnectionsSummary token={token} organizationId={organizationId} />
        </aside>
      </div>

      <OlderRuns token={token} organizationId={organizationId} projects={projects} />
    </>
  );
}
