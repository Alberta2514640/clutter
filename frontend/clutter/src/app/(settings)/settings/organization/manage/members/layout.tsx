import type { Metadata } from "next";

export const metadata: Metadata = { title: "Team members" };

export default function TeamMembersLayout({ children }: { children: React.ReactNode }) {
  return children;
}
