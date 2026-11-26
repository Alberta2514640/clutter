import { WELCOME_PATH } from "@/lib/features/organization/home";

// The four onboarding screens, in order. Step 2 keeps its older address,
// which links and the end-to-end test know.
export const ONBOARDING_STEPS = [WELCOME_PATH, "/onboarding/create-org", "/onboarding/cloud", "/onboarding/start"] as const;

export const [EXPERIENCE_PATH, TEAM_PATH, CLOUD_PATH, START_PATH] = ONBOARDING_STEPS;

// stepNumber is the 1-based step of an onboarding address, or 1 for any other.
export function stepNumber(pathname: string | null): number {
  const index = ONBOARDING_STEPS.indexOf((pathname ?? "") as (typeof ONBOARDING_STEPS)[number]);
  return index < 0 ? 1 : index + 1;
}
