"use client";

import ErrorState from "@/components/common/ErrorState";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { ArrowLeft, ArrowRight, Loader2 } from "lucide-react";
import Link from "next/link";
import type { FormEvent, ReactNode } from "react";

type Props = {
  title: string;
  description?: ReactNode;
  // The id the question's options name themselves by (aria-labelledby).
  titleId?: string;
  children: ReactNode;
  onContinue: () => void;
  continueLabel?: string;
  continueDisabled?: boolean;
  pending?: boolean;
  pendingLabel?: string;
  backHref?: string;
  // A quiet way past an optional step.
  skip?: { label: string; onSkip: () => void };
  // A refusal from the last attempt, shown above the buttons.
  error?: string | null;
  // A small block drawing above the question (StepArt). Decorative.
  illustration?: ReactNode;
};

// StepFrame is one onboarding question: the heading, its options, then Back
// on the left and Skip and Continue on the right. It is a form, so Enter in a
// field continues.
export default function StepFrame({
  title,
  description,
  titleId,
  children,
  onContinue,
  continueLabel = "Continue",
  continueDisabled = false,
  pending = false,
  pendingLabel = "Working…",
  backHref,
  skip,
  error,
  illustration,
}: Props) {
  const submit = (event: FormEvent) => {
    event.preventDefault();
    if (!continueDisabled && !pending) onContinue();
  };

  return (
    <form onSubmit={submit} noValidate>
      {/* A fixed height, so a drawing that grows with the answer moves nothing. */}
      {illustration && (
        <div aria-hidden="true" className="mb-6 flex h-14 items-end gap-3">
          {illustration}
        </div>
      )}
      <h1 id={titleId} className="text-foreground text-2xl font-semibold tracking-tight text-balance md:text-3xl">
        {title}
      </h1>
      {description && <p className="text-muted-foreground mt-2 text-base">{description}</p>}

      <div className="mt-8">{children}</div>

      {error && (
        <p role="alert" className="border-danger/25 bg-danger/10 text-foreground animate-fade-in mt-6 rounded-lg border p-3 text-sm">
          {error}
        </p>
      )}

      <div className="mt-10 flex flex-col-reverse gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          {backHref && (
            <Button asChild variant="outline">
              <Link href={backHref}>
                <ArrowLeft aria-hidden="true" />
                Back
              </Link>
            </Button>
          )}
        </div>
        <div className="flex flex-col-reverse items-stretch gap-3 sm:flex-row sm:items-center">
          {skip && (
            <Button type="button" variant="ghost" onClick={skip.onSkip} disabled={pending} className="text-muted-foreground">
              {skip.label}
            </Button>
          )}
          <Button type="submit" size="lg" disabled={continueDisabled || pending} aria-disabled={continueDisabled || pending}>
            {pending ? (
              <>
                <Loader2 className="animate-spin" aria-hidden="true" />
                {pendingLabel}
              </>
            ) : (
              <>
                {continueLabel}
                <ArrowRight aria-hidden="true" />
              </>
            )}
          </Button>
        </div>
      </div>
    </form>
  );
}

// StepLoading holds a step's place while the session and teams load.
export function StepLoading() {
  return (
    <div role="status" className="space-y-4">
      <span className="sr-only">Loading…</span>
      <Skeleton className="h-9 w-96 max-w-full" />
      <Skeleton className="h-5 w-72 max-w-full" />
      <div className="space-y-3 pt-6">
        <Skeleton className="h-[72px] w-full rounded-xl" />
        <Skeleton className="h-[72px] w-full rounded-xl" />
        <Skeleton className="h-[72px] w-full rounded-xl" />
      </div>
    </div>
  );
}

// StepError says the teams could not load, with Retry.
export function StepError({ onRetry, retrying }: { onRetry: () => void; retrying: boolean }) {
  return (
    <ErrorState
      title="Clutter could not load your account"
      description="Check your connection, then try again."
      onRetry={onRetry}
      retrying={retrying}
    />
  );
}
