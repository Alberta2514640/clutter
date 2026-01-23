"use client";
import { useProjectActions, useProjectState } from "@/lib/stores/projectStore";
import { useRunActions, useRunState } from "@/lib/stores/runStore";
import { useUserActions, useUserState, useUserStore } from "@/lib/stores/userStore";
import { useEffect } from "react";
import DashboardContent from "./_components/DashboardContent";
import DashboardLoading from "./_components/DashboardLoading";
// import DashboardOnboarding from "./_components/DashboardOnboarding";

export default function DashboardPageClient() {
  // Access state from stores
  const userState = useUserState();
  const projectState = useProjectState();
  const runState = useRunState();

  // Access actions from stores
  const userActions = useUserActions();
  const projectActions = useProjectActions();
  const runActions = useRunActions();

  // MOVED: Define function before useEffect
  const fetchDashboardData = async () => {
    // Load user first
    await userActions.loadUser();

    // If user has a tenant, load projects and runs
    const currentUser = useUserStore.getState().state.user;
    if (currentUser?.tenantId) {
      await Promise.all([
        projectActions.loadProjects(),
        runActions.loadRecentRuns(),
      ]);
    }
  };

  // Load data on mount
  useEffect(() => {
    fetchDashboardData();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Determine loading state (any store loading)
  const isLoading = userState.isLoading || projectState.isLoading || runState.isLoading;

  // Combine errors from all stores
  const error = userState.error || projectState.error || runState.error;

  if (isLoading && !userState.user) return <DashboardLoading />;
  // if (userState.user && !userState.user.tenantId) return <DashboardOnboarding />;

  return (
    <DashboardContent
      userData={userState.user}
      projects={projectState.projects}
      recentRuns={runState.runs}
      error={error}
    />
  );
}