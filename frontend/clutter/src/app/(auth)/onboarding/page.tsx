"use client";

import { RadioCard, RadioCardDescription, RadioCardGroup, RadioCardTitle } from "@/components/ui/radio-card";
import { usePreferencesStore, type Experience } from "@/lib/features/preferences/uiStore";
import { useIsClient } from "@/lib/features/user/useIsClient";
import { Code2, Compass, MousePointerClick, type LucideIcon } from "lucide-react";
import { useRouter } from "next/navigation";
import { useId, useState } from "react";
import { ExperienceArt } from "./_components/StepArt";
import StepFrame, { StepError, StepLoading } from "./_components/StepFrame";
import { TEAM_PATH } from "./_components/steps";
import { useOnboardingSession } from "./_components/useOnboardingSession";

// Step 1 of 4: how much the person has done with cloud setups. It is kept in
// this browser only and changes defaults: the first-deploy checklist open
// for the first two, one-line hints on blocks for the first only, and the
// code preview open for the third.
const OPTIONS: { value: Experience; title: string; description: string; icon: LucideIcon }[] = [
  { value: "new", title: "I'm new to this. Show me the way.", description: "Clutter keeps the first-deploy checklist open and explains each block.", icon: Compass },
  { value: "console", title: "I've used a cloud console, not code.", description: "You know the parts. Clutter writes the code for you.", icon: MousePointerClick },
  { value: "terraform", title: "I write Terraform or similar code.", description: "The code preview opens by default, so you can read what Clutter writes.", icon: Code2 },
];

const isExperience = (v: string): v is Experience => OPTIONS.some((o) => o.value === v);

export default function ExperienceStep() {
  const router = useRouter();
  const session = useOnboardingSession("before-team");
  const isClient = useIsClient();
  const stored = usePreferencesStore((s) => s.experience);
  const setExperience = usePreferencesStore((s) => s.setExperience);
  const [picked, setPicked] = useState<Experience | null>(null);
  const titleId = useId();
  const value = picked ?? (isClient ? stored : null);

  if (session.isError) return <StepError onRetry={session.retry} retrying={session.retrying} />;
  if (!session.ready) return <StepLoading />;

  const next = (experience: Experience | null) => {
    setExperience(experience);
    router.push(TEAM_PATH);
  };

  return (
    <StepFrame
      title="How much have you worked with cloud setups?"
      titleId={titleId}
      illustration={<ExperienceArt value={value} />}
      description="This sets a few defaults. You can change it later in your preferences."
      onContinue={() => value && next(value)}
      continueDisabled={!value}
      skip={{ label: "Skip", onSkip: () => next(null) }}>
      <RadioCardGroup
        aria-labelledby={titleId}
        value={value ?? ""}
        onValueChange={(v) => isExperience(v) && setPicked(v)}
        onEnter={() => value && next(value)}>
        {OPTIONS.map(({ value: v, title, description, icon: Icon }) => (
          <RadioCard key={v} value={v} data-testid={`onboarding-experience-${v}`}>
            <span className="bg-muted text-foreground group-data-[state=checked]/radio-card:bg-primary/10 group-data-[state=checked]/radio-card:text-link flex size-10 shrink-0 items-center justify-center rounded-lg transition-colors">
              <Icon className="size-5" aria-hidden="true" />
            </span>
            <span className="min-w-0">
              <RadioCardTitle>{title}</RadioCardTitle>
              <RadioCardDescription>{description}</RadioCardDescription>
            </span>
          </RadioCard>
        ))}
      </RadioCardGroup>
    </StepFrame>
  );
}
