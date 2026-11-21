"use client";

import { useCurrentOrganization } from "@/lib/features/organization/hooks";
import { useMe } from "@/lib/features/user/hooks";
import Members from "../_components/Members";

// Team settings, Members.
export default function TeamMembersPage() {
  const meQ = useMe();
  const current = useCurrentOrganization(meQ.data?.token ?? null);
  if (!current.organization) return null;
  return <Members me={meQ.data ?? null} memberCount={current.organization.total_members} />;
}
