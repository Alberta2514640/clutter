'use client';

import { useEffect, useState } from 'react';
import { apiClient } from '@/lib/api-client';
import DashboardLoading from './_components/DashboardLoading';
import DashboardOnboarding from './_components/DashboardOnboarding';
import DashboardContent from './_components/DashboardContent';

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

export default function DashboardPage() {
  const [userData, setUserData] = useState<UserData | null>(null);
  const [projects, setProjects] = useState<Project[]>([]);
  const [recentRuns, setRecentRuns] = useState<Run[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchDashboardData();
  }, []);

  const fetchDashboardData = async () => {
    try {
      setLoading(true);
      setError(null);

      const user = await apiClient.getUserProfile();
      setUserData(user);

      if (user.tenantId) {
        const [projectsData, runsData] = await Promise.all([
          apiClient.getProjects(),
          apiClient.getRecentRuns()
        ]);

        setProjects(projectsData.projects || []);
        setRecentRuns(runsData.runs || []);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred');
    } finally {
      setLoading(false);
    }
  };

  // State 1: Loading
  if (loading) {
    return <DashboardLoading />;
  }

  // State 2: No Tenant (Onboarding)
  if (userData && !userData.tenantId) {
    return <DashboardOnboarding />;
  }

  // State 3: Dashboard Content
  return (
    <DashboardContent
      userData={userData}
      projects={projects}
      recentRuns={recentRuns}
      error={error}
    />
  );
}