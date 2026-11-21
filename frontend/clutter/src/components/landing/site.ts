// Facts the public pages (home, privacy, terms) and their metadata share.

// The site's own origin, for canonical links, the sitemap and Open Graph URLs.
// In the cloud the website and B API share one origin (ADR-023), and the deploy
// builds the site with NEXT_PUBLIC_ADAPTER_API_ENDPOINT = https://<site_domain>,
// so that address is the site's origin. Anything else (a local build) falls
// back to the local front door.
function siteOrigin(): string {
  const adapter = process.env.NEXT_PUBLIC_ADAPTER_API_ENDPOINT;
  if (adapter) {
    try {
      const url = new URL(adapter);
      if (url.protocol === "https:") return url.origin;
    } catch {
      // Not a URL; use the local address.
    }
  }
  return "http://localhost:3000";
}

export const SITE_URL = siteOrigin();

// Domains reserved for examples and tests (RFC 2606 and RFC 6761). A local
// build copies hello@example.com from .env.example; the pages treat it as no
// address at all, as backend/infra/check-config.sh does for the cloud.
const RESERVED_DOMAIN = /(^|\.)(example\.(com|net|org)|example|test|invalid|localhost)$/i;

// The address people write to. It is a build setting (frontend/clutter/.env.example,
// and contact_email in the platform's tfvars for the cloud), never text in code.
// A missing, malformed or example value gives null, and the pages say so.
export function contactEmail(raw: string | undefined): string | null {
  const v = raw?.trim();
  if (!v || !/^[^\s@<>"]+@[^\s@<>"]+\.[^\s@<>"]+$/.test(v)) return null;
  return RESERVED_DOMAIN.test(v.split("@")[1]) ? null : v;
}

export const CONTACT_EMAIL = contactEmail(process.env.NEXT_PUBLIC_CONTACT_EMAIL);

// Who runs Clutter, as the privacy and terms pages name it.
export const OPERATOR = "Alberta 2514640 Inc.";

// The date both legal pages were last changed. Change it with their text.
export const LEGAL_UPDATED = "2026-11-13";
export const LEGAL_UPDATED_TEXT = "November 13, 2026";

export const HOME_TITLE = "Clutter: one diagram, every cloud you link";
export const HOME_DESCRIPTION =
  "Draw your infrastructure once in plain building blocks. Clutter writes the Terraform and builds it on AWS and OpenStack, inside the accounts you link.";

// One focus style for every link and button on the public pages, from the tokens.
export const FOCUS_RING = "outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background";
