import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { useHasOlderDiagrams } from "@/lib/features/diagram/hooks";
import { useProjectRecentActivity } from "@/lib/features/logs/hooks";
import { Organization } from "@/lib/features/organization/types";
import { useProjects } from "@/lib/features/projects/hooks";
import { useMe } from "@/lib/features/user/hooks";
import { AlertCircle } from "lucide-react";
import ActivitySection from "./ActivitySection";
import DashboardHeader from "./DashboardHeader";
import ProjectsSection from "./ProjectSection";
import StatsCards from "./StatsCards";

interface DashboardContentProps {
  organization: Organization | null;
  error: string | null;
}

export default function DashboardContent({ organization, error }: DashboardContentProps) {
  const meQ = useMe();

  const token = meQ.data?.token ?? null;

  const projectsQ = useProjects(token, organization?.id);
  const projects = projectsQ.data ?? [];

  const { data: recentActivity = [], isLoading } = useProjectRecentActivity(token, organization?.id, projects);
  // Only A's older AWS-only diagrams report runs here; B's show theirs on the diagram's Deploy tab.
  const older = useHasOlderDiagrams(
    token,
    projects.map((p) => p.id),
  );

  return (
    <div className="px-4 py-8 md:px-6 md:py-12">
      <DashboardHeader organizationName={organization?.name} />

      {/* Without the organization, the counts and first steps below would be wrong, so only the error shows. */}
      {error ? (
        <Alert className="mb-6 border-red-800 bg-red-900/20 text-red-200">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle className="text-red-100">Something went wrong</AlertTitle>
          <AlertDescription className="text-red-200">{error}</AlertDescription>
        </Alert>
      ) : (
        <>
          <StatsCards projectCount={projectsQ.data?.length ?? 0} diagramCount={organization?.total_diagrams ?? 0} memberCount={organization?.total_members ?? 0} />

          <ProjectsSection projects={projects} isLoading={projectsQ.isLoading} isError={projectsQ.isError} onRetry={() => void projectsQ.refetch()} />
          {older.hasOlder && <ActivitySection activity={recentActivity} isLoading={isLoading} />}
        </>
      )}
    </div>
  );
}
