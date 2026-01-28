// proxy.ts (project root)
import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";

const PUBLIC = ["/login", "/terms", "/privacy", "/favicon.ico", "/robots.txt"];

//new draft for proxy not used rn, proxy needs cookies so changes need to be made before using this
export async function proxy(req: NextRequest) {
  // const { pathname, origin } = req.nextUrl;

  // const isPublic =
  //   PUBLIC.some((p) => pathname.startsWith(p)) ||
  //   pathname.startsWith("/_next") ||
  //   pathname.startsWith("/api") ||
  //   pathname.startsWith("/logos");

  // const token = req.cookies.get("clutter_token")?.value;

  // if (!token) {
  //   if (!isPublic) return NextResponse.redirect(new URL("/login", origin));
  //   return NextResponse.next();
  // }

  // // allow onboarding page itself
  // if (pathname.startsWith("/onboarding")) return NextResponse.next();

  // // Check org membership (server-side)
  // try {
  //   const orgRes = await fetch(new URL("/api/organization/list", origin), {
  //     headers: { cookie: req.headers.get("cookie") ?? "" },
  //     cache: "no-store",
  //   });

  //   if (!orgRes.ok) return NextResponse.next();

  //   const orgs = (await orgRes.json()) as unknown[];
  //   const hasOrg = Array.isArray(orgs) && orgs.length > 0;

  //   if (!hasOrg) {
  //     return NextResponse.redirect(new URL("/onboarding/create-org", origin));
  //   }

  //   return NextResponse.next();
  // } catch {
  //   return NextResponse.next();
  // }

  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico|robots.txt).*)"],
};
