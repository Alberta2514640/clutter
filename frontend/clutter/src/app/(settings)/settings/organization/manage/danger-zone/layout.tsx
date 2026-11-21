import type { Metadata } from "next";

export const metadata: Metadata = { title: "Danger zone" };

export default function TeamDangerZoneLayout({ children }: { children: React.ReactNode }) {
  return children;
}
