import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import type { Project } from "@/lib/features/projects/types";
import type { Run } from "@/lib/features/runs/types";
import type { UserData } from "@/lib/features/user/types";
import { AlertCircle } from "lucide-react";
import ActivitySection from "./ActivitySection";
import DashboardHeader from "./DashboardHeader";
import ProjectsSection from "./ProjectSection";
import StatsCards from "./StatsCards";

interface DashboardContentProps {
  userData: UserData | null;
  projects: Project[];
  recentRuns: Run[];
  error: string | null;
}

export default function DashboardContent({ userData, projects, recentRuns, error }: DashboardContentProps) {
  return (
    <div className="px-6 py-12">
      <DashboardHeader tenantName={userData?.tenant?.name} />

      {error && (
        <Alert className="mb-6 bg-red-900/20 border-red-800">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Error</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <StatsCards projectCount={projects.length} runCount={recentRuns.length} memberCount={projects.reduce((acc, p) => acc + (p.memberCount || 0), 0)} />

      <ProjectsSection projects={projects} />

      <ActivitySection runs={recentRuns} />
    </div>
  );
}
