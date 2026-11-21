import { cn } from "@/lib/utils";
import type { ReactNode } from "react";
import AppSidebar, { MobileNav } from "./Sidebar";

// The one page width and gutter for every app page. AppShell applies it;
// export it only for a full-width band that must line up with the content.
const PAGE_CONTAINER = "mx-auto w-full max-w-6xl px-4 md:px-8";

// AppShell is every signed-in page except the editor: the dark sidebar and
// the content column. The content follows the viewer's theme, over the app
// backdrop and its grain, which stay fixed while the content scrolls.
export default function AppShell({ children }: { children: ReactNode }) {
  return (
    <div className="bg-background text-foreground relative isolate flex min-h-dvh">
      <div aria-hidden className="bg-app-backdrop grain pointer-events-none fixed inset-0 -z-10" />
      <a
        href="#main"
        className="focus:bg-primary focus:text-primary-foreground sr-only focus:not-sr-only focus:fixed focus:top-3 focus:left-3 focus:z-[60] focus:rounded-md focus:px-4 focus:py-2 focus:text-sm focus:font-medium">
        Skip to content
      </a>
      <AppSidebar />
      <div className="flex min-w-0 flex-1 flex-col">
        <MobileNav />
        <main id="main" tabIndex={-1} className="flex-1 outline-none">
          <div className={cn(PAGE_CONTAINER, "py-6 md:py-10")}>{children}</div>
        </main>
      </div>
    </div>
  );
}
