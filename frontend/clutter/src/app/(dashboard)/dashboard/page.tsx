"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";

import { useOrganizations } from "@/lib/features/organization/hooks";
import DashboardContent from "./_components/DashboardContent";
import DashboardLoading from "./_components/DashboardLoading";

import { useProjects } from "@/lib/features/projects/hooks";
import { useRecentRuns } from "@/lib/features/runs/hooks";
import { useMe } from "@/lib/features/user/hooks";

export default function DashboardPageClient() {
  const router = useRouter();

  const meQ = useMe();
  const token = meQ.data?.token ?? null;

  const orgQ = useOrganizations(token);
  const organization = orgQ.data?.[0] ?? null;

  const projectsQ = useProjects(token);
  const runsQ = useRecentRuns(token);
  //  redirect logic after render
  useEffect(() => {
    if (meQ.isLoading) return;

    if (!token) {
      router.replace("/login");
      return;
    }

    if (!orgQ.isLoading && !orgQ.isError && !organization) {
      router.replace("/onboarding/create-org");
    }
  }, [meQ.isLoading, token, orgQ.isLoading, orgQ.isError, organization, router]);

  const isLoading = meQ.isLoading || (token ? orgQ.isLoading : false) || (token ? runsQ.isLoading : false);

  const error = (meQ.isError ? meQ.error : null) || (orgQ.isError ? orgQ.error : null) || (runsQ.isError ? runsQ.error : null);

  if (isLoading && !meQ.data) return <DashboardLoading />;

  // While redirecting (no org), avoid flashing dashboard
  if (token && !orgQ.isLoading && !organization) return <DashboardLoading />;

  return <DashboardContent userData={meQ.data ?? null} organization={organization} recentRuns={runsQ.data ?? []} error={error ? String(error) : null} />;
}
