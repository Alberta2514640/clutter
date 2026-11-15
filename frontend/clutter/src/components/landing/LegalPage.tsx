import { LandingHeader } from "./LandingHeader";
import { SiteFooter } from "./SiteFooter";
import { LEGAL_UPDATED, LEGAL_UPDATED_TEXT } from "./site";

// The frame of the privacy and terms pages: header, a readable column, footer.
export function LegalPage({ title, intro, children }: { title: string; intro: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen flex-col bg-black text-white">
      <LandingHeader />
      <main id="main" className="flex-1">
        <article className="mx-auto max-w-3xl px-4 py-12 md:px-6 md:py-16">
          <h1 className="text-3xl font-bold tracking-tight md:text-4xl">{title}</h1>
          <p className="mt-3 text-sm text-slate-400">
            Last updated: <time dateTime={LEGAL_UPDATED}>{LEGAL_UPDATED_TEXT}</time>
          </p>
          <div className="mt-6 text-lg leading-relaxed text-slate-300">{intro}</div>
          <div className="mt-10 space-y-10 leading-relaxed text-slate-300 [&_h2]:mb-3 [&_h2]:text-xl [&_h2]:font-semibold [&_h2]:text-white [&_li]:mt-2 [&_p+p]:mt-3 [&_strong]:text-slate-100 [&_ul]:mt-3 [&_ul]:list-disc [&_ul]:pl-6">
            {children}
          </div>
        </article>
      </main>
      <SiteFooter />
    </div>
  );
}
