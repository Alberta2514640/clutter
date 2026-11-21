import ClutterLogo from "@/components/common/ClutterLogo";
import NotFoundPanel from "@/components/common/NotFoundPanel";
import { Button } from "@/components/ui/button";
import { PROJECTS_PATH } from "@/lib/features/organization/home";
import type { Metadata } from "next";
import Link from "next/link";

export const metadata: Metadata = { title: "Page not found" };

// Any address that matches no page. Signed-out visitors never see it: the
// route guard (src/proxy.ts) sends them to /login first.
export default function NotFound() {
  return (
    <div className="bg-background text-foreground flex min-h-dvh flex-col">
      <header className="mx-auto flex h-16 w-full max-w-6xl items-center px-4 md:px-8">
        <Link href="/" aria-label="Clutter home" className="rounded-md outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50">
          <ClutterLogo size={28} withName />
        </Link>
      </header>
      <main className="flex flex-1 items-center justify-center pb-16">
        <NotFoundPanel
          actions={
            <>
              <Button asChild>
                <Link href={PROJECTS_PATH}>Go to Projects</Link>
              </Button>
              <Button asChild variant="outline">
                <Link href="/">Go to the home page</Link>
              </Button>
            </>
          }
        />
      </main>
    </div>
  );
}
