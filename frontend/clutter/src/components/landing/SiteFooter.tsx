import ClutterLogo from "@/components/common/ClutterLogo";
import ThemeScope from "@/components/common/ThemeScope";
import Link from "next/link";
import { ContactEmail } from "./ContactEmail";
import { CONTACT_EMAIL, FOCUS_RING, OPERATOR } from "./site";

const link = `hover:text-foreground rounded underline-offset-4 hover:underline ${FOCUS_RING}`;

// The public pages' footer, always dark: the product, who runs it, contact
// (only when this build has an address), and the legal pages.
export function SiteFooter() {
  return (
    <ThemeScope theme="dark" className="bg-hero">
      <footer className="border-border border-t">
        <div className="text-muted-foreground mx-auto flex max-w-6xl flex-col gap-8 px-4 py-12 text-sm md:flex-row md:items-start md:justify-between md:px-8">
          <div className="space-y-3">
            <ClutterLogo size={28} withName />
            <p>Draw your infrastructure once. Deploy it to the clouds you link.</p>
            {CONTACT_EMAIL && (
              <p>
                Questions? Write to <ContactEmail />.
              </p>
            )}
          </div>
          <div className="space-y-3 md:text-right">
            <nav aria-label="Legal">
              <ul className="flex gap-6 md:justify-end">
                <li>
                  <Link href="/privacy" className={link}>
                    Privacy
                  </Link>
                </li>
                <li>
                  <Link href="/terms" className={link}>
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
    </ThemeScope>
  );
}
