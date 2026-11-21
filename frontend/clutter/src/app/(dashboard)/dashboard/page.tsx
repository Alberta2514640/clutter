"use client";

import ErrorState from "@/components/common/ErrorState";
import PageSkeleton from "@/components/common/PageSkeleton";
import { homePath } from "@/lib/features/organization/home";
import { useCurrentOrganization } from "@/lib/features/organization/hooks";
import { useMe } from "@/lib/features/user/hooks";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import ProjectsHome from "./_components/ProjectsHome";

// Projects: the current team's home. A person without a team goes to Welcome.
export default function ProjectsPage() {
  const router = useRouter();
  const meQ = useMe();
  const token = meQ.data?.token ?? null;
  const current = useCurrentOrganization(token);

  useEffect(() => {
    if (meQ.isLoading) return;
    if (!token) {
      router.replace("/login");
      return;
    }
    if (current.isSuccess && !current.organization) router.replace(homePath(current.organizations));
  }, [meQ.isLoading, token, current.isSuccess, current.organization, current.organizations, router]);

  if (current.isError) {
    return (
      <ErrorState
        title="Your teams could not load"
        description="Check your connection, then try again."
        onRetry={() => void current.refetch()}
        retrying={current.isFetching}
      />
    );
  }
  if (!token || !current.organization) return <PageSkeleton label="Loading your projects…" />;

  return <ProjectsHome token={token} organization={current.organization} />;
}
