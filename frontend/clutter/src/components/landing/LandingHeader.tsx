import ClutterLogo from "@/components/common/ClutterLogo";
import { Button } from "@/components/ui/button";
import Link from "next/link";
import { HeaderBar } from "./HeaderBar";
import { SectionsMenu, type SectionLink } from "./SectionsMenu";
import { FOCUS_RING } from "./site";

const sections: SectionLink[] = [
  { href: "/#demo", label: "Demo" },
  { href: "/#security", label: "Security" },
  { href: "/#examples", label: "Examples" },
  { href: "/#clouds", label: "Clouds" },
  { href: "/#faq", label: "FAQ" },
];

// The public pages' header, always dark so the wordmark reads on the hero:
// the logo, links to the home page's sections (behind a Menu button below
// the md width), and Sign in (left out on the sign-in page itself). overHero lays it over the home page's hero, which
// then starts 4rem (the header's height) higher.
export function LandingHeader({ signIn = true, overHero = false }: { signIn?: boolean; overHero?: boolean }) {
  return (
    <HeaderBar overHero={overHero}>
      <header>
        <a
          href="#main"
          className="bg-primary text-primary-foreground sr-only rounded-md px-4 py-2 font-medium focus:not-sr-only focus:absolute focus:top-3 focus:left-4 focus:z-50">
          Skip to content
        </a>
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between gap-4 px-4 md:px-8">
          <Link href="/" aria-label="Clutter, home" className={`rounded ${FOCUS_RING}`}>
            <ClutterLogo size={30} withName />
          </Link>
          <nav aria-label="Sections" className="hidden md:block">
            <ul className="text-muted-foreground flex items-center gap-1 text-sm">
              {sections.map((s) => (
                <li key={s.href}>
                  <Link href={s.href} className={`hover:text-foreground rounded-md px-3 py-2 transition-colors ${FOCUS_RING}`}>
                    {s.label}
                  </Link>
                </li>
              ))}
            </ul>
          </nav>
          <div className="flex items-center gap-2">
            <SectionsMenu sections={sections} />
            {signIn ? (
              <Button asChild size="sm" className="h-9 px-4">
                <Link href="/login">Sign in</Link>
              </Button>
            ) : (
              <span className="hidden w-[78px] md:block" aria-hidden="true" />
            )}
          </div>
        </div>
      </header>
    </HeaderBar>
  );
}
