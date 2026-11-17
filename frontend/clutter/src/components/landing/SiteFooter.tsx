import Link from "next/link";
import { ContactEmail } from "./ContactEmail";
import { FOCUS_RING, OPERATOR } from "./site";

// The public pages' footer: the product, who runs it, contact, and the legal pages.
export function SiteFooter() {
  return (
    <footer className="border-t border-slate-800/80 bg-black">
      <div className="mx-auto flex max-w-6xl flex-col gap-6 px-4 py-10 text-sm text-slate-400 md:flex-row md:items-start md:justify-between md:px-6">
        <div className="space-y-2">
          <p className="text-base font-semibold text-white">Clutter</p>
          <p>Draw your infrastructure once. Deploy it to the clouds you link.</p>
          <p>
            Questions? Write to <ContactEmail />.
          </p>
        </div>
        <div className="space-y-3 md:text-right">
          <nav aria-label="Legal">
            <ul className="flex gap-4 md:justify-end">
              <li>
                <Link href="/privacy" className={`rounded underline-offset-4 hover:text-white hover:underline ${FOCUS_RING}`}>
                  Privacy
                </Link>
              </li>
              <li>
                <Link href="/terms" className={`rounded underline-offset-4 hover:text-white hover:underline ${FOCUS_RING}`}>
                  Terms
                </Link>
              </li>
            </ul>
          </nav>
          <p>
            © {new Date().getFullYear()} {OPERATOR}
          </p>
        </div>
      </div>
    </footer>
  );
}
