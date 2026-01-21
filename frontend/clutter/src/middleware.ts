import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";

export async function middleware(req: NextRequest) {
  const { pathname, origin } = req.nextUrl;

  const isPublic =
    pathname === "/login" ||
    pathname.startsWith("/onboarding") ||
    pathname.startsWith("/api");

  // if you store a session cookie/JWT, check it here
  const hasAuth = req.cookies.get("session") || req.cookies.get("token");

  if (!hasAuth) {
    // block protected routes
    if (!isPublic && pathname !== "/") {
      return NextResponse.redirect(new URL("/login", origin));
    }
    return NextResponse.next();
  }

  // user is logged in: if trying to access protected routes, enforce tenant check
  if (!isPublic && pathname !== "/") {
    const meRes = await fetch(new URL("/api/me", origin), {
      headers: { cookie: req.headers.get("cookie") ?? "" },
    });

    if (meRes.ok) {
      const me = await meRes.json();
      if (!me.tenantId) {
        return NextResponse.redirect(
          new URL("/onboarding/create-tenant", origin),
        );
      }
    }
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
