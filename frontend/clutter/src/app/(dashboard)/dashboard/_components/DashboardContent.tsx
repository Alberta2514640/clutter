import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Organization } from "@/lib/features/organization/types";
import { useProjects } from "@/lib/features/projects/hooks";
import { useMe } from "@/lib/features/user/hooks";
import type { UserData } from "@/lib/features/user/types";
import { AlertCircle } from "lucide-react";
import ActivitySection from "./ActivitySection";
import DashboardHeader from "./DashboardHeader";
import ProjectsSection from "./ProjectSection";
import StatsCards from "./StatsCards";


//runs is a placeholder for now it should be changed in the near future
interface DashboardContentProps {
  userData: UserData | null;
  organization: Organization | null;
  error: string | null;
}

export default function DashboardContent({ userData, organization, error }: DashboardContentProps) {
  const meQ = useMe();

  const token = meQ.data?.token ?? null;

  const projectsQ = useProjects(token, organization?.id);
  const projects = projectsQ.data ?? [];

  return (
    <div className="px-6 py-12">
      <DashboardHeader userName={userData?.displayName} />

      {error && (
        <Alert className="mb-6 bg-red-900/20 border-red-800">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Error</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <StatsCards
        projectCount={projectsQ.data?.length ?? 0}
        diagramCount={organization?.total_diagrams ?? 0}
        // need to firgue out the team memebr situation dont quite know how to handle that yet
        memberCount={organization?.total_members ?? 0}
      />

      <ProjectsSection projects={projects} />
      <ActivitySection token={token} organizationId={organization?.id ?? null} projects={projects} />
    </div>
  );
}
