"use client";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "@/components/ui/sonner";
import { useCreateOrganization } from "@/lib/features/organization/hooks";
import { defaultTeamName, organizationNameHint, organizationNameProblem, organizationNameTaken } from "@/lib/features/organization/rules";
import { isConflict } from "@/lib/features/user/session";
import { useRouter } from "next/navigation";
import { useRef, useState } from "react";
import { TeamArt } from "../_components/StepArt";
import StepFrame, { StepError, StepLoading } from "../_components/StepFrame";
import { CLOUD_PATH, EXPERIENCE_PATH } from "../_components/steps";
import { useOnboardingSession } from "../_components/useOnboardingSession";

// Step 2 of 4: the person's first team (an organization in the code, the API
// and the address). The name starts as "<first name>'s team".
export default function TeamStep() {
  const router = useRouter();
  const session = useOnboardingSession("before-team");
  const createOrg = useCreateOrganization(session.token);
  const inputRef = useRef<HTMLInputElement>(null);
  // null until the person types: the suggestion follows the profile until then.
  const [typed, setTyped] = useState<string | null>(null);
  const [showProblem, setShowProblem] = useState(false);

  if (session.isError) return <StepError onRetry={session.retry} retrying={session.retrying} />;
  if (!session.ready) return <StepLoading />;

  const name = typed ?? defaultTeamName(session.me?.displayName);
  const problem = organizationNameProblem(name);
  const shownProblem = showProblem ? problem : null;
  const conflict = createOrg.isError && isConflict(createOrg.error);
  const fieldError = shownProblem ?? (conflict ? organizationNameTaken : null);
  const otherError = createOrg.isError && !conflict ? "Clutter could not create the team. Try again." : null;

  const create = async () => {
    if (problem) {
      setShowProblem(true);
      inputRef.current?.focus();
      return;
    }
    session.leave();
    try {
      await createOrg.mutateAsync({ organizationName: name.trim() });
      toast.success("Team created");
      router.push(CLOUD_PATH);
    } catch {
      // Shown from createOrg.error above the buttons or under the field.
    }
  };

  return (
    <StepFrame
      title="What should we call your team?"
      illustration={<TeamArt />}
      description="A team holds your projects, diagrams and cloud connections. You can rename it later in Settings."
      onContinue={() => void create()}
      pending={createOrg.isPending}
      pendingLabel="Creating your team…"
      backHref={EXPERIENCE_PATH}
      error={otherError}>
      <div className="space-y-2">
        <Label htmlFor="onboarding-team-name">Team name</Label>
        <Input
          ref={inputRef}
          id="onboarding-team-name"
          name="team-name"
          value={name}
          onChange={(e) => {
            setTyped(e.target.value);
            if (createOrg.isError) createOrg.reset();
          }}
          onBlur={() => setShowProblem(true)}
          autoComplete="organization"
          autoFocus
          aria-invalid={!!fieldError}
          aria-describedby="onboarding-team-name-hint"
          className="h-11 text-base md:text-base"
        />
        <p id="onboarding-team-name-hint" role={fieldError ? "alert" : undefined} className={fieldError ? "text-danger text-sm" : "text-muted-foreground text-sm"}>
          {fieldError ?? organizationNameHint}
        </p>
      </div>
    </StepFrame>
  );
}
