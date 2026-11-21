"use client";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { AlertTriangle, Loader2, RotateCw } from "lucide-react";
import type { ReactNode } from "react";

type Props = {
  title?: string;
  // One plain sentence about what failed. Never the raw error.
  description: ReactNode;
  onRetry?: () => void;
  // True while the retry runs: the button says so and waits.
  retrying?: boolean;
  retryLabel?: string;
  // Another way out, such as a link back to Projects.
  action?: ReactNode;
  // "panel" for a page or a section, "inline" for a small row (a list in the sidebar).
  variant?: "panel" | "inline";
  className?: string;
};

// ErrorState says what could not load, in a plain sentence, with Retry.
export default function ErrorState({
  title = "Something went wrong",
  description,
  onRetry,
  retrying = false,
  retryLabel = "Retry",
  action,
  variant = "panel",
  className,
}: Props) {
  const retry = onRetry && (
    <Button type="button" variant="outline" size={variant === "inline" ? "xs" : "sm"} onClick={onRetry} disabled={retrying}>
      {retrying ? <Loader2 className="animate-spin" aria-hidden="true" /> : <RotateCw aria-hidden="true" />}
      {retrying ? "Retrying…" : retryLabel}
    </Button>
  );

  if (variant === "inline") {
    return (
      <div role="alert" className={cn("flex flex-wrap items-center gap-2 text-sm", className)}>
        <AlertTriangle className="text-danger size-4 shrink-0" aria-hidden="true" />
        <span className="text-muted-foreground min-w-0 flex-1">{description}</span>
        {retry}
      </div>
    );
  }

  return (
    <div role="alert" className={cn("surface-card px-6 py-10 text-center animate-fade-in", className)}>
      <span className="bg-danger/10 text-danger mx-auto flex size-10 items-center justify-center rounded-lg">
        <AlertTriangle className="size-5" aria-hidden="true" />
      </span>
      <h2 className="text-foreground mt-4 text-base font-semibold">{title}</h2>
      <p className="text-muted-foreground mx-auto mt-1.5 max-w-md text-sm">{description}</p>
      {(retry || action) && (
        <div className="mt-6 flex flex-wrap items-center justify-center gap-2">
          {retry}
          {action}
        </div>
      )}
    </div>
  );
}
