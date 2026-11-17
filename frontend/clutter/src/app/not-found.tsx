import ClutterLogo from "@/components/common/ClutterLogo";
import type { Metadata } from "next";
import Link from "next/link";

export const metadata: Metadata = { title: "Page not found" };

// Any address that matches no page. Signed-out visitors never see it: the
// route guard (src/proxy.ts) sends them to /login first.
export default function NotFound() {
  return (
    <main className="flex min-h-screen items-center justify-center bg-gradient-to-br from-black via-slate-900 to-teal-900 px-4 text-white">
      <div className="w-full max-w-md rounded-2xl border border-slate-800 bg-slate-950/70 p-8 text-center shadow-2xl">
        <Link href="/" aria-label="Clutter home" className="inline-flex rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-teal-300">
          <ClutterLogo size={40} withName />
        </Link>
        <p className="mt-8 text-sm font-semibold uppercase tracking-wider text-teal-300">404</p>
        <h1 className="mt-2 text-2xl font-semibold">Page not found</h1>
        <p className="mt-2 text-sm text-slate-400">This address does not match any page. Check the link, or go back to a page you know.</p>
        <div className="mt-8 flex flex-col justify-center gap-3 sm:flex-row">
          <Link
            href="/dashboard"
            className="inline-flex items-center justify-center rounded-md bg-teal-600 px-4 py-2 text-sm font-medium text-white hover:bg-teal-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-teal-300">
            Go to the dashboard
          </Link>
          <Link
            href="/"
            className="inline-flex items-center justify-center rounded-md border border-slate-700 px-4 py-2 text-sm font-medium text-slate-200 hover:bg-slate-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-teal-300">
            Go to the home page
          </Link>
        </div>
      </div>
    </main>
  );
}
