import ClutterLogo from "@/components/common/ClutterLogo";
import Link from "next/link";
import { FOCUS_RING } from "./site";

const sections = [
  { href: "/#demo", label: "Demo" },
  { href: "/#how-it-works", label: "How it works" },
  { href: "/#clouds", label: "Clouds" },
  { href: "/#trust", label: "Security" },
  { href: "/#faq", label: "FAQ" },
];

// The public pages' header: the logo, links to the home page's sections, and
// Sign in (left out on the sign-in page itself).
export function LandingHeader({ signIn = true }: { signIn?: boolean }) {
  return (
    <header className="sticky top-0 z-50 border-b border-white/5 bg-[#020514]/80 backdrop-blur-md">
      <a
        href="#main"
        className="sr-only rounded-md bg-teal-300 px-4 py-2 font-medium text-black focus:not-sr-only focus:absolute focus:left-4 focus:top-3 focus:z-50">
        Skip to content
      </a>
      <div className="mx-auto flex h-16 max-w-6xl items-center justify-between gap-4 px-4 md:px-6">
        <Link href="/" className={`rounded ${FOCUS_RING}`}>
          <ClutterLogo size={32} withName />
        </Link>
        <nav aria-label="Sections" className="hidden md:block">
          <ul className="flex items-center gap-1 text-sm text-slate-300">
            {sections.map((s) => (
              <li key={s.href}>
                <Link href={s.href} className={`rounded-md px-3 py-2 hover:text-white ${FOCUS_RING}`}>
                  {s.label}
                </Link>
              </li>
            ))}
          </ul>
        </nav>
        {signIn ? (
          <Link
            href="/login"
            className={`rounded-md border border-cyan-400/50 px-4 py-2 text-sm font-medium text-cyan-200 transition-colors hover:border-cyan-300 hover:text-white ${FOCUS_RING}`}>
            Sign in
          </Link>
        ) : (
          <span className="hidden w-[78px] md:block" aria-hidden="true" />
        )}
      </div>
    </header>
  );
}
