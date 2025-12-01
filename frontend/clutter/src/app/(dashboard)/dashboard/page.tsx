"use client";

import { useAuth } from "@/lib/hooks/queries/useAuth";
import { useProjects } from "@/lib/hooks/queries/useProject";
import { useRecentRuns } from "@/lib/hooks/queries/useRecentRuns";
import DashboardContent from "./_components/DashboardContent";
import DashboardLoading from "./_components/DashboardLoading";

export default function DashboardPageClient() {
  // Get user data from React Query
  const { data: user, isLoading: userLoading } = useAuth();

  // Get projects - only fetch if user exists
  const { 
    data: projectsData, 
    isLoading: projectsLoading 
  } = useProjects(user?.organizationId || "", {
    enabled: !!user?.organizationId, // Only fetch if organizationId exists
  });

  // Get recent runs - only fetch if user exists
  const { 
    data: runsData, 
    isLoading: runsLoading 
  } = useRecentRuns({
    enabled: !!user?.organizationId, // Only fetch if organizationId exists
  });

  // Show loading while fetching initial data
  if (userLoading) return <DashboardLoading />;

  // Show dashboard with data
  // Note: organizationId is always present since it's created on first login
  return (
    <DashboardContent
      userData={{
        userId: user?.userId || "",
        organizationId: user?.organizationId || "",
        email: user?.email || "",
        displayName: user?.displayName || "",
        pictureUrl: user?.pictureUrl || "",
        createdAt: user?.createdAt || "",
      }}
      projects={projectsData?.projects || []}
      recentRuns={runsData?.runs || []}
      error={null}
    />
  );
}