// proxy.ts (project root)
import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";

// The public pages and what they load without a session: the legal pages, the
// sitemap, the Open Graph image (crawlers have no cookie), the site icon and the
// landing images.
const PUBLIC = ["/login", "/favicon.ico", "/robots.txt", "/privacy", "/terms", "/sitemap.xml", "/opengraph-image", "/icon.svg", "/landing/", "/clouds/", "/guide/"];

export function proxy(req: NextRequest) {
  const { pathname, origin } = req.nextUrl;

  // ✅ allow homepage for everyone
  if (pathname === "/") return NextResponse.next();

  const isPublic =
    PUBLIC.some((p) => pathname.startsWith(p)) ||
    pathname.startsWith("/_next") ||
    pathname.startsWith("/api") ||
    pathname.startsWith("/logos");

  const token = req.cookies.get("clutter_token")?.value;

  // ✅ if not logged in, only allow public routes
  if (!token && !isPublic) {
    return NextResponse.redirect(new URL("/login", origin));
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico|robots.txt).*)"],
};
