import type { Metadata } from "next";

export const metadata: Metadata = { title: "New team" };

export default function CreateTeamLayout({ children }: { children: React.ReactNode }) {
  return children;
}
