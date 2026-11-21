import type { Metadata } from "next";
import TeamGeneral from "./_components/TeamGeneral";

export const metadata: Metadata = { title: "Team settings" };

export default function TeamGeneralPage() {
  return <TeamGeneral />;
}
