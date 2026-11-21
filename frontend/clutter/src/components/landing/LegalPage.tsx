import ThemeScope from "@/components/common/ThemeScope";
import { LandingHeader } from "./LandingHeader";
import { SiteFooter } from "./SiteFooter";
import { LEGAL_UPDATED, LEGAL_UPDATED_TEXT } from "./site";

// The frame of the privacy and terms pages: the dark header and footer, and a
// readable light column between them.
export function LegalPage({ title, intro, children }: { title: string; intro: React.ReactNode; children: React.ReactNode }) {
  return (
    <ThemeScope theme="light" className="bg-background text-foreground flex min-h-screen flex-col">
      <LandingHeader />
      <main id="main" className="flex-1">
        <article className="mx-auto max-w-3xl px-4 py-12 md:px-8 md:py-20">
          <h1 className="text-4xl font-bold tracking-tight md:text-5xl">{title}</h1>
          <p className="text-muted-foreground mt-4 text-sm">
            Last updated: <time dateTime={LEGAL_UPDATED}>{LEGAL_UPDATED_TEXT}</time>
          </p>
          <div className="text-muted-foreground mt-8 text-lg leading-relaxed">{intro}</div>
          <div className="text-muted-foreground [&_h2]:text-foreground [&_strong]:text-foreground mt-12 space-y-12 leading-relaxed [&_h2]:mb-4 [&_h2]:text-2xl [&_h2]:font-semibold [&_h2]:tracking-tight [&_li]:mt-2 [&_p+p]:mt-3 [&_strong]:font-semibold [&_ul]:mt-3 [&_ul]:list-disc [&_ul]:pl-6">
            {children}
          </div>
        </article>
      </main>
      <SiteFooter />
    </ThemeScope>
  );
}
