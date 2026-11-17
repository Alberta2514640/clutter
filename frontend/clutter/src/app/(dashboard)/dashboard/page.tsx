"use client";

import { useCurrentOrganization } from "@/lib/features/organization/hooks";
import { useMe } from "@/lib/features/user/hooks";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import DashboardContent from "./_components/DashboardContent";
import DashboardLoading from "./_components/DashboardLoading";

export default function DashboardPageClient() {
  const router = useRouter();

  const meQ = useMe();
  const token = meQ.data?.token ?? null;

  const orgQ = useCurrentOrganization(token);
  const organization = orgQ.organization;

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

  const isLoading = meQ.isLoading || (token ? orgQ.isLoading : false);

  const error = meQ.isError || orgQ.isError ? "Clutter could not load your organization. Reload the page to try again." : null;

  if (isLoading && !meQ.data) return <DashboardLoading />;

  // While redirecting (no org), avoid flashing dashboard
  if (token && !orgQ.isLoading && !orgQ.isError && !organization) return <DashboardLoading />;

  return <DashboardContent organization={organization} error={error} />;
}
