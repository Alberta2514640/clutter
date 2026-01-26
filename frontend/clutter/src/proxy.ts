import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";

export function proxy(req: NextRequest) {
  // const { pathname } = req.nextUrl;

  // -----------------------------
  // Auth gate (optional)
  // -----------------------------
  // const publicRoutes = ["/login", "/"];

  // const isPublic =
  //   publicRoutes.includes(pathname) ||
  //   pathname.startsWith("/onboarding") ||
  //   pathname.startsWith("/api") ||
  //   pathname.startsWith("/logos") ||
  //   pathname.startsWith("/_next") ||
  //   pathname === "/favicon.ico" ||
  //   pathname === "/robots.txt" ||
  //   pathname === "/sitemap.xml" ||
  //   pathname === "/terms" ||
  //   pathname === "/privacy";

  // const hasSession =
  //   req.cookies.has("session") || req.cookies.has("token");

  // if (!hasSession && !isPublic) {
  //   const loginUrl = req.nextUrl.clone();
  //   loginUrl.pathname = "/login";
  //   return NextResponse.redirect(loginUrl);
  // }

  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
