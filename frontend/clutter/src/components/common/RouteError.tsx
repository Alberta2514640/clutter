"use client";

import { Button } from "@/components/ui/button";
import { PROJECTS_PATH } from "@/lib/features/organization/home";
import Link from "next/link";
import ErrorState from "./ErrorState";

// RouteError is the body of every group's error.tsx: a plain sentence, Retry
// (Next's retry re-fetches and re-renders the segment), and a way back. The
// error itself is never shown; Next logs it in development.
export default function RouteError({ retry, backHref = PROJECTS_PATH, backLabel = "Go to Projects" }: { retry: () => void; backHref?: string; backLabel?: string }) {
  return (
    <ErrorState
      title="This page could not load"
      description="Something went wrong while Clutter was showing this page. Try again, or go back."
      onRetry={retry}
      action={
        <Button asChild variant="ghost" size="sm">
          <Link href={backHref}>{backLabel}</Link>
        </Button>
      }
    />
  );
}
