"use client";

import { themeScript } from "@/components/common/theme";
import "./globals.css";

// An error in the root layout itself. It replaces the whole document, so it
// brings its own <html>, styles and theme script, and uses no providers.
export default function GlobalError({ retry }: { error: unknown; retry: () => void }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <title>Something went wrong | Clutter</title>
        <script dangerouslySetInnerHTML={{ __html: themeScript }} />
      </head>
      <body className="bg-background text-foreground font-sans antialiased">
        <main className="grid min-h-dvh place-items-center px-4">
          <div role="alert" className="surface-card w-full max-w-md p-8 text-center">
            <h1 className="text-xl font-semibold tracking-tight">Clutter could not load</h1>
            <p className="text-muted-foreground mt-2 text-sm">Something went wrong before the page could show. Try again, or go to the home page.</p>
            <div className="mt-6 flex flex-col justify-center gap-3 sm:flex-row">
              <button
                type="button"
                onClick={() => retry()}
                className="bg-primary text-primary-foreground hover:bg-primary/90 focus-visible:ring-ring/50 inline-flex h-9 items-center justify-center rounded-md px-4 text-sm font-medium outline-none focus-visible:ring-[3px]">
                Retry
              </button>
              {/* A plain link: the router may be what failed. */}
              {/* eslint-disable-next-line @next/next/no-html-link-for-pages */}
              <a
                href="/"
                className="border-input hover:bg-accent focus-visible:ring-ring/50 inline-flex h-9 items-center justify-center rounded-md border px-4 text-sm font-medium outline-none focus-visible:ring-[3px]">
                Go to the home page
              </a>
            </div>
          </div>
        </main>
      </body>
    </html>
  );
}
