"use client";

import RouteError from "@/components/common/RouteError";

// An unexpected error on sign-in or onboarding. The way back is the sign-in page.
export default function AuthError({ retry }: { error: unknown; retry: () => void }) {
  return (
    <div className="grid min-h-dvh place-items-center px-4">
      <div className="w-full max-w-md">
        <RouteError retry={retry} backHref="/login" backLabel="Go to sign in" />
      </div>
    </div>
  );
}
