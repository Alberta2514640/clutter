"use client";

import { homePath } from "@/lib/features/organization/home";
import { useCurrentOrganization } from "@/lib/features/organization/hooks";
import { useMe } from "@/lib/features/user/hooks";
import { useRouter } from "next/navigation";
import { useEffect, useRef } from "react";
import { TEAM_PATH } from "./steps";

// Who may see an onboarding step. Steps 1 and 2 come before the team exists:
// a person who already has a team goes where they start (homePath). Steps 3
// and 4 need the team step 2 made: without one, they go back to step 2. A
// step that is about to move on calls leave(), so the team it just created
// does not send it to Projects first.
export function useOnboardingSession(stage: "before-team" | "after-team") {
  const router = useRouter();
  const meQ = useMe();
  const token = meQ.data?.token ?? null;
  const current = useCurrentOrganization(token);
  const leaving = useRef(false);
  const hasTeam = current.organizations.length > 0;

  useEffect(() => {
    if (meQ.isLoading) return;
    if (!token) {
      router.replace("/login");
      return;
    }
    if (!current.isSuccess || leaving.current) return;
    if (stage === "before-team" && hasTeam) router.replace(homePath(current.organizations));
    if (stage === "after-team" && !hasTeam) router.replace(TEAM_PATH);
  }, [meQ.isLoading, token, current.isSuccess, current.organizations, hasTeam, stage, router]);

  return {
    token,
    me: meQ.data ?? null,
    organization: current.organization,
    isError: meQ.isError || current.isError,
    retry: () => void current.refetch(),
    retrying: current.isFetching,
    // The step may show: signed in, the teams loaded, and on the right side of step 2.
    ready: !!token && current.isSuccess && (stage === "before-team" ? !hasTeam : hasTeam),
    leave: () => {
      leaving.current = true;
    },
  };
}
