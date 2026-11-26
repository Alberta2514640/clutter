"use client";

import { ThemeProvider } from "@/components/common/ThemeProvider";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { isSessionExpired, expireSession } from "@/lib/features/user/session";
import { MutationCache, QueryCache, QueryClient, QueryClientProvider } from "@tanstack/react-query";
import * as React from "react";
// Optional devtools
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";

// Every query and mutation passes its error here, so an expired session sends
// the user to the login page from any screen, A's or B's.
const onAnyError = (err: unknown) => {
  if (isSessionExpired(err)) expireSession();
};

export function Providers({ children }: { children: React.ReactNode }) {
  //  create once per browser session
  const [queryClient] = React.useState(
    () =>
      new QueryClient({
        queryCache: new QueryCache({ onError: onAnyError }),
        mutationCache: new MutationCache({ onError: onAnyError }),
        defaultOptions: {
          queries: {
            // Data counts as fresh for a minute, returning to the window does
            // not refetch, and a failed query retries once unless the session expired.
            staleTime: 60_000,
            refetchOnWindowFocus: false,
            retry: (failureCount, err) => !isSessionExpired(err) && failureCount < 1,
          },
        },
      }),
  );

  // The theme, one tooltip delay and one toaster for every screen.
  return (
    <ThemeProvider>
      <QueryClientProvider client={queryClient}>
        <TooltipProvider>
          {children}
          <Toaster />
        </TooltipProvider>
        <ReactQueryDevtools initialIsOpen={false} />
      </QueryClientProvider>
    </ThemeProvider>
  );
}
