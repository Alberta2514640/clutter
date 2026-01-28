"use client";

import { useOrganizations } from "@/lib/features/organization/hooks";
import DashboardContent from "./_components/DashboardContent";
import DashboardLoading from "./_components/DashboardLoading";
// import DashboardOnboarding from "./_components/DashboardOnboarding";

import { useProjects } from "@/lib/features/projects/hooks";
import { useRecentRuns } from "@/lib/features/runs/hooks";
import { useMe } from "@/lib/features/user/hooks";

export default function DashboardPageClient() {
  const meQ = useMe();
  const token = meQ.data?.token ?? null;
  const projectsQ = useProjects(token);

  const orgQ = useOrganizations(token);
  const organization = orgQ.data?.[0] ?? null;

  const runsQ = useRecentRuns(token);

  const isLoading =
    meQ.isLoading ||
    (token ? runsQ.isLoading : false);

  const error =
    (meQ.isError ? meQ.error : null) ||
    (runsQ.isError ? runsQ.error : null);

  if (isLoading && !meQ.data) return <DashboardLoading />;

  // If you want onboarding logic:
  // if (meQ.data && !meQ.data.tenantId) return <DashboardOnboarding />;

  return (
    <DashboardContent
      userData={meQ.data ?? null}
      organization={organization ?? null}
      recentRuns={runsQ.data ?? []}
      error={error ? String(error) : null}
    />
  );
}
