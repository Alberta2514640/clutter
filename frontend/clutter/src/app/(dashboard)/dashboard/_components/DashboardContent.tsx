import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { AlertCircle } from 'lucide-react';
import DashboardHeader from './DashboardHeader';
import StatsCards from './StatsCards';
import ProjectsSection from './ProjectSection';
import ActivitySection from './ActivitySection';

interface Project {
  projectId: string;
  name: string;
  description: string;
  createdAt: string;
  updatedAt: string;
  memberCount?: number;
}

interface Run {
  runId: string;
  projectId: string;
  projectName: string;
  workspaceId: string;
  action: 'plan' | 'apply';
  status: 'QUEUED' | 'RUNNING' | 'SUCCESS' | 'FAILED';
  startedAt: string;
  endedAt?: string;
}

interface UserData {
  userId: string;
  tenantId: string | null;
  email: string;
  displayName: string;
  tenant?: {
    tenantId: string;
    name: string;
  };
}

interface DashboardContentProps {
  userData: UserData | null;
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
      <DashboardHeader tenantName={userData?.tenant?.name} />

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