"use client";

import { useCurrentOrganization } from "@/lib/features/organization/hooks";
import { useMe } from "@/lib/features/user/hooks";
import TeamGeneralForm from "./TeamGeneralForm";

// Team settings, General: the team's name and description.
export default function TeamGeneral() {
  const meQ = useMe();
  const token = meQ.data?.token ?? null;
  const current = useCurrentOrganization(token);
  const team = current.organization;
  if (!team) return null;

  const otherNames = current.organizations.filter((o) => o.id !== team.id).map((o) => o.name);
  // Keyed by the saved values, so a save or a switch of team starts the form again.
  return <TeamGeneralForm key={`${team.id}:${team.name}:${team.description ?? ""}`} team={team} token={token} otherNames={otherNames} />;
}
