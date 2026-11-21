import type { Metadata } from "next";

export const metadata: Metadata = { title: "Account and preferences" };

export default function AccountLayout({ children }: { children: React.ReactNode }) {
  return children;
}
