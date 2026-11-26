"use client";

import ErrorState from "@/components/common/ErrorState";
import NavTabs from "@/components/common/NavTabs";
import NotFoundPanel from "@/components/common/NotFoundPanel";
import PageHeader from "@/components/common/PageHeader";
import PageSkeleton from "@/components/common/PageSkeleton";
import { Button } from "@/components/ui/button";
import { PROJECTS_PATH } from "@/lib/features/organization/home";
import { useOrganizations } from "@/lib/features/organization/hooks";
import { useProjectOrganization } from "@/lib/features/projects/hooks";
import { useMe } from "@/lib/features/user/hooks";
import { useQueryClient } from "@tanstack/react-query";
import { ChevronLeft } from "lucide-react";
import Link from "next/link";
import { useParams, usePathname } from "next/navigation";
import { useMemo } from "react";
import NewDiagramButton from "./_components/NewDiagramDialog";
import { ProjectProvider, useProjectContext } from "./_contexts/ProjectContext";

interface ProjectLayoutProps {
  children: React.ReactNode;
}

const backToProjects = (
  <Button asChild>
    <Link href={PROJECTS_PATH}>Back to Projects</Link>
  </Button>
);

// A project id that is not one of the viewer's (deleted, mistyped, or another
// team's) gets a way back, not a blank page.
function ProjectNotFound() {
  return (
    <NotFoundPanel
      title="Project not found"
      description="This project does not exist, or it belongs to a team you are not in."
      actions={backToProjects}
    />
  );
}

// Retry refetches every query on the page that failed.
function useRetryFailed() {
  const queryClient = useQueryClient();
  return () => queryClient.refetchQueries({ predicate: (query) => query.state.status === "error" });
}

function ProjectLayoutContent({ children }: ProjectLayoutProps) {
  const { isLoading, hasError, project, projectId } = useProjectContext();
  const pathname = usePathname() ?? "";
  const retry = useRetryFailed();

  if (isLoading && !project) return <PageSkeleton label="Loading the project…" />;
  if (hasError && !project) {
    return (
      <ErrorState
        title="This project could not load"
        description="Check your connection, then try again."
        onRetry={retry}
        action={
          <Button asChild variant="ghost" size="sm">
            <Link href={PROJECTS_PATH}>Back to Projects</Link>
          </Button>
        }
      />
    );
  }
  if (!project) return <ProjectNotFound />;

  const base = `/projects/${projectId}`;
  // The Diagrams tab's empty state holds the one New diagram button.
  // The root address counts as the Diagrams tab: its server redirect lands
  // after this layout first renders, and the header must not flicker.
  const onDiagramsTab = pathname === base || pathname.startsWith(`${base}/diagrams`);
  const onEmptyDiagrams = onDiagramsTab && (project.diagrams?.length ?? 0) === 0;

  return (
    <div className="flex min-w-0 flex-col">
      <PageHeader
        eyebrow={
          <Link href={PROJECTS_PATH} className="hover:text-foreground inline-flex min-h-6 items-center gap-1 rounded-sm outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50">
            <ChevronLeft className="size-3.5" aria-hidden="true" />
            Projects
          </Link>
        }
        title={project.name}
        description={project.description?.trim() || undefined}
        // Hidden, not unmounted: /projects/[id] redirects to /diagrams, and
        // unmounting here would close a dialog opened in that moment.
        actions={
          <div className={onEmptyDiagrams ? "hidden" : undefined}>
            <NewDiagramButton openFromLink />
          </div>
        }>
        <NavTabs
          label="Project"
          tabs={[
            { href: `${base}/diagrams`, label: "Diagrams" },
            { href: `${base}/deployments`, label: "Deployments" },
            { href: `${base}/settings`, label: "Settings" },
          ]}
        />
      </PageHeader>
      <div key={pathname} className="animate-fade-in">
        {children}
      </div>
    </div>
  );
}

export default function ProjectLayout({ children }: ProjectLayoutProps) {
  const params = useParams<{ projectId: string }>();
  const projectId = params.projectId;
  const retry = useRetryFailed();

  const meQ = useMe();
  const token = meQ.data?.token ?? null;

  const orgQ = useOrganizations(token);
  const orgIds = useMemo(() => orgQ.data?.map((org) => org.id) ?? null, [orgQ.data]);
  // The project's own organization, not the user's first one: the backend
  // refuses a project named under another organization.
  const projectOrgQ = useProjectOrganization(token, projectId, orgIds);
  const organizationId = projectOrgQ.data ?? null;

  if (!projectId) return <ProjectNotFound />;

  if (meQ.isError || orgQ.isError || projectOrgQ.isError) {
    return <ErrorState title="This project could not load" description="Check your connection, then try again." onRetry={retry} />;
  }

  const noOrganizations = orgQ.isSuccess && (orgIds?.length ?? 0) === 0;
  if (noOrganizations || (projectOrgQ.isSuccess && !organizationId)) return <ProjectNotFound />;

  if (!organizationId) return <PageSkeleton label="Loading the project…" />;

  return (
    <ProjectProvider organizationId={organizationId} projectId={projectId}>
      <ProjectLayoutContent>{children}</ProjectLayoutContent>
    </ProjectProvider>
  );
}
