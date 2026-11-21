"use client";

import { DotGrid } from "@/components/common/CanvasMotif";
import ClutterLogo from "@/components/common/ClutterLogo";
import { toast } from "@/components/ui/sonner";
import { useLogout, useMe } from "@/lib/features/user/hooks";
import { cn } from "@/lib/utils";
import { usePathname, useRouter } from "next/navigation";
import { ONBOARDING_STEPS, stepNumber } from "./steps";

// Onboarding's frame: a light page with a faint dot grid, the logo top left,
// the segmented progress top right, one question per screen in a centred
// column, and a way to sign out at the foot.
function Progress({ step, total }: { step: number; total: number }) {
  return (
    <div className="flex flex-col items-end gap-2">
      <div
        role="progressbar"
        aria-label="Onboarding progress"
        aria-valuemin={1}
        aria-valuemax={total}
        aria-valuenow={step}
        aria-valuetext={`Step ${step} of ${total}`}
        className="flex gap-1.5">
        {Array.from({ length: total }, (_, i) => (
          <span key={i} className="bg-border relative h-1.5 w-8 overflow-hidden rounded-full sm:w-10">
            <span
              className={cn(
                "bg-primary motion-slow ease-standard absolute inset-0 origin-left rounded-full transition-transform",
                i < step ? "scale-x-100" : "scale-x-0",
              )}
            />
          </span>
        ))}
      </div>
      <p aria-hidden="true" className="text-muted-foreground text-xs font-medium">
        Step {step} of {total}
      </p>
    </div>
  );
}

export default function OnboardingFrame({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const meQ = useMe();
  const logout = useLogout();
  const step = stepNumber(pathname);

  const signOut = () => {
    logout();
    toast.success("You are signed out.");
    router.push("/login");
    router.refresh();
  };

  return (
    <div className="relative isolate flex min-h-dvh flex-col">
      <DotGrid fade="top" className="-z-10" />
      <header className="mx-auto flex w-full max-w-5xl items-start justify-between gap-4 px-4 pt-6 sm:px-8">
        <ClutterLogo size={28} withName />
        <Progress step={step} total={ONBOARDING_STEPS.length} />
      </header>

      <main id="main" className="flex-1 px-4 sm:px-8">
        {/* Each step slides and fades in as it opens. */}
        <div
          key={pathname}
          className="animate-in fade-in-0 slide-in-from-right-4 motion-slow ease-standard mx-auto w-full max-w-[720px] pt-12 pb-16 md:pt-20">
          {children}
        </div>
      </main>

      <footer className="text-muted-foreground mx-auto flex w-full max-w-5xl flex-wrap items-center justify-center gap-x-2 gap-y-1 px-4 pb-6 text-xs sm:justify-start sm:px-8">
        {meQ.data?.email && <span>Signed in as {meQ.data.email}.</span>}
        <button
          type="button"
          onClick={signOut}
          className="text-foreground rounded-sm font-medium underline-offset-4 outline-none hover:underline focus-visible:ring-[3px] focus-visible:ring-ring/50">
          Sign out
        </button>
      </footer>
    </div>
  );
}
