"use client";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { useOrganizations } from "@/lib/features/organization/hooks";
import { useProjectOrganization } from "@/lib/features/projects/hooks";
import { useMe } from "@/lib/features/user/hooks";
import { Loader2 } from "lucide-react";
import { useParams } from "next/navigation";
import { useMemo } from "react";
import { ProjectHeader } from "./_components/ProjectHeader";
import { ProjectNav } from "./_components/ProjectNav";
import { ProjectProvider, useProjectContext } from "./_contexts/ProjectContext";

interface ProjectLayoutProps {
  children: React.ReactNode;
}

/**
 * Inner layout component that uses the project context
 */
function ProjectLayoutContent({ children }: ProjectLayoutProps) {
  const { isLoading, hasError, project } = useProjectContext();
  
  return (
    <div className="flex min-w-0 flex-1 flex-col">
      <ProjectHeader />
      <ProjectNav />

      <main className="flex-1 px-4 py-8 md:px-12">
        {/* Loading state */}
        {isLoading && !project && (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="w-8 h-8 animate-spin text-teal-400" />
          </div>
        )}
        
        {/* Error state */}
        {hasError && !project && (
          <Alert className="bg-red-900/20 border-red-800">
            <AlertDescription>
              Clutter could not load this project. Reload the page to try again.
            </AlertDescription>
          </Alert>
        )}
        
        {/* Content */}
        {!isLoading && project && children}
        
        {/* Not found state */}
        {!isLoading && !project && !hasError && (
          <div className="flex items-center justify-center py-12 text-gray-400">
            Project not found
          </div>
        )}
      </main>
    </div>
  );
}

/**
 * Main layout component that provides the project context
 */
export default function ProjectLayout({ children }: ProjectLayoutProps) {
  const params = useParams<{ projectId: string }>();
  const projectId = params.projectId;

  const meQ = useMe();
  const token = meQ.data?.token ?? null;

  const orgQ = useOrganizations(token);
  const orgIds = useMemo(() => orgQ.data?.map((org) => org.id) ?? null, [orgQ.data]);
  // The project's own organization, not the user's first one: the backend
  // refuses a project named under another organization.
  const projectOrgQ = useProjectOrganization(token, projectId, orgIds);
  const organizationId = projectOrgQ.data ?? null;

  if (!projectId) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <p className="text-gray-400">No project ID provided</p>
      </div>
    );
  }

  if (meQ.isError || orgQ.isError || projectOrgQ.isError) {
    return (
      <div className="flex items-center justify-center min-h-screen px-12">
        <Alert className="bg-red-900/20 border-red-800">
          <AlertDescription>Failed to load project. Reload the page to try again.</AlertDescription>
        </Alert>
      </div>
    );
  }

  const noOrganizations = orgQ.isSuccess && (orgIds?.length ?? 0) === 0;
  if (noOrganizations || (projectOrgQ.isSuccess && !organizationId)) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <p className="text-gray-400">Project not found</p>
      </div>
    );
  }

  if (!organizationId) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <Loader2 className="w-8 h-8 animate-spin text-teal-400" />
      </div>
    );
  }

  return (
    <div className="flex min-h-screen bg-gradient-to-br from-black via-slate-900 to-teal-900 text-white">
      <ProjectProvider organizationId={organizationId} projectId={projectId}>
        <ProjectLayoutContent>{children}</ProjectLayoutContent>
      </ProjectProvider>
    </div>
  );
}