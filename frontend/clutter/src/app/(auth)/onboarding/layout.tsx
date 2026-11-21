import type { Metadata } from "next";
import OnboardingFrame from "./_components/OnboardingFrame";

export const metadata: Metadata = { title: "Welcome" };

export default function OnboardingLayout({ children }: { children: React.ReactNode }) {
  return <OnboardingFrame>{children}</OnboardingFrame>;
}
