import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { AlertCircle } from 'lucide-react';
import DashboardHeader from './DashboardHeader';
import StatsCards from './StatsCards';
import ProjectsSection from './ProjectSection';
import ActivitySection from './ActivitySection';
import type { Run, User } from "@/lib/types";


interface Project {
  projectId: string;
  name: string;
  description: string;
  createdAt: string;
  updatedAt: string;
  memberCount?: number;
}





interface DashboardContentProps {
  userData: User;
  projects: Project[];
  recentRuns: Run[];
  error: string | null;
}

export default function DashboardContent({
  userData,
  projects,
  recentRuns,
  error
}: DashboardContentProps) {
  return (
    <div className="px-6 py-12">
      <DashboardHeader
        organizationId={userData?.organizationId}
        tenantName={userData?.organizationId} />

      {error && (
        <Alert className="mb-6 bg-red-900/20 border-red-800">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Error</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <StatsCards 
        projectCount={projects.length}
        runCount={recentRuns.length}
        memberCount={projects.reduce((acc, p) => acc + (p.memberCount || 0), 0)}
      />

      <ProjectsSection projects={projects} />

      <ActivitySection runs={recentRuns} />
    </div>
  );
}