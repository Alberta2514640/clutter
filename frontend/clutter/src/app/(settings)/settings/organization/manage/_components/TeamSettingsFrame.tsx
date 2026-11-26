"use client";

import EmptyState from "@/components/common/EmptyState";
import ErrorState from "@/components/common/ErrorState";
import NavTabs from "@/components/common/NavTabs";
import PageHeader from "@/components/common/PageHeader";
import PageSkeleton from "@/components/common/PageSkeleton";
import { Button } from "@/components/ui/button";
import { useCurrentOrganization } from "@/lib/features/organization/hooks";
import { useMe } from "@/lib/features/user/hooks";
import { Users } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";

const TEAM_SETTINGS_HREF = "/settings/organization/manage";
const NEW_TEAM_HREF = "/settings/organization/create-org";

// Team settings (the current organization): General, Members and Danger
// zone, each its own address under one header. The team is the one chosen in
// the sidebar's switcher.
export default function TeamSettingsFrame({ children }: { children: React.ReactNode }) {
  const pathname = usePathname() ?? "";
  const meQ = useMe();
  const current = useCurrentOrganization(meQ.data?.token ?? null);

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
  if (meQ.isLoading || current.isLoading) return <PageSkeleton label="Loading team settings…" cards={2} />;

  if (!current.organization) {
    return (
      <>
        <PageHeader eyebrow="Settings" title="Team settings" />
        <EmptyState
          icon={Users}
          title="Create a team first"
          description="Settings are about a team: its name, its members, and deleting it. You are not in a team yet."
          action={
            <Button asChild>
              <Link href={NEW_TEAM_HREF}>Create a team</Link>
            </Button>
          }
        />
      </>
    );
  }

  return (
    <>
      <PageHeader eyebrow="Settings" title="Team settings" description={`For ${current.organization.name}. Switch teams in the sidebar.`}>
        <NavTabs
          label="Team settings"
          tabs={[
            { href: TEAM_SETTINGS_HREF, label: "General", isActive: (p) => p === TEAM_SETTINGS_HREF },
            { href: `${TEAM_SETTINGS_HREF}/members`, label: "Members" },
            { href: `${TEAM_SETTINGS_HREF}/danger-zone`, label: "Danger zone" },
          ]}
        />
      </PageHeader>
      <div key={pathname} className="animate-fade-in max-w-3xl">
        {children}
      </div>
    </>
  );
}
