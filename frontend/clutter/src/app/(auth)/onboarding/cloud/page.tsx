"use client";

import CloudLogo from "@/components/common/CloudLogo";
import { RadioCard, RadioCardDescription, RadioCardGroup, RadioCardTitle } from "@/components/ui/radio-card";
import { NOT_SURE_CLOUD, usePreferencesStore } from "@/lib/features/preferences/uiStore";
import { useIsClient } from "@/lib/features/user/useIsClient";
import { HelpCircle } from "lucide-react";
import { useRouter } from "next/navigation";
import { useId, useState } from "react";
import { CloudArt } from "../_components/StepArt";
import StepFrame, { StepError, StepLoading } from "../_components/StepFrame";
import { START_PATH } from "../_components/steps";
import { useOnboardingSession } from "../_components/useOnboardingSession";

// Step 3 of 4: the cloud to start with, kept in this browser only. It orders
// step 4's examples, sets the editor's "View as" default and opens that
// cloud's form when the checklist's Connect step runs. "Not sure yet" is kept
// too, so Back shows it chosen, and changes nothing else.
const OPTIONS = [
  { value: "aws", title: "AWS", description: "Amazon Web Services, in your own account" },
  { value: "general-openstack", title: "OpenStack", description: "VEXXHOST and other OpenStack clouds" },
  { value: NOT_SURE_CLOUD, title: "Not sure yet", description: "Try it first, decide later" },
] as const;

export default function CloudStep() {
  const router = useRouter();
  const session = useOnboardingSession("after-team");
  const isClient = useIsClient();
  const stored = usePreferencesStore((s) => s.firstCloud);
  const setFirstCloud = usePreferencesStore((s) => s.setFirstCloud);
  const [picked, setPicked] = useState<string | null>(null);
  const titleId = useId();
  const storedValue = isClient && stored && OPTIONS.some((o) => o.value === stored) ? stored : null;
  const value = picked ?? storedValue;

  if (session.isError) return <StepError onRetry={session.retry} retrying={session.retrying} />;
  if (!session.ready) return <StepLoading />;

  const next = () => {
    if (!value) return;
    setFirstCloud(value);
    router.push(START_PATH);
  };

  // Back is not offered: step 2 made the team, and that is not undone here.
  // Skip, like the other steps' Skip, keeps no choice and moves on.
  return (
    <StepFrame
      title="Which cloud do you want to start with?"
      titleId={titleId}
      illustration={<CloudArt connector={value && value !== NOT_SURE_CLOUD ? value : null} />}
      description="You can connect more clouds any time."
      onContinue={next}
      continueDisabled={!value}
      skip={{ label: "Skip", onSkip: () => router.push(START_PATH) }}>
      <RadioCardGroup aria-labelledby={titleId} value={value ?? ""} onValueChange={setPicked} onEnter={next}>
        {OPTIONS.map((option) => (
          <RadioCard key={option.value} value={option.value} data-testid={`onboarding-cloud-${option.value}`}>
            <span className="bg-background flex size-12 shrink-0 items-center justify-center rounded-lg border">
              {option.value === NOT_SURE_CLOUD ? (
                <HelpCircle className="text-muted-foreground size-6" aria-hidden="true" />
              ) : (
                <CloudLogo connector={option.value} size={28} />
              )}
            </span>
            <span className="min-w-0">
              <RadioCardTitle>{option.title}</RadioCardTitle>
              <RadioCardDescription>{option.description}</RadioCardDescription>
            </span>
          </RadioCard>
        ))}
      </RadioCardGroup>
    </StepFrame>
  );
}
