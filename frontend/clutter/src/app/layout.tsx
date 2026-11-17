// app/layout.tsx

import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import { SITE_URL } from "@/components/landing/site";
import "./globals.css";
import { Providers } from "./providers";

// The two typefaces, self-hosted by next/font at build time. Their CSS
// variables feed --font-sans and --font-mono in globals.css.
const geistSans = Geist({ subsets: ["latin"], display: "swap", variable: "--font-geist-sans" });
// Not preloaded: only code and logs in the app use it, and preloading it on
// every page cost the landing page's phone score.
const geistMono = Geist_Mono({ subsets: ["latin"], display: "swap", variable: "--font-geist-mono", preload: false });

// Defaults for every page. The signed-in app is not indexed; the public pages
// (/, /privacy, /terms) set robots to index in their own metadata.
export const metadata: Metadata = {
  metadataBase: new URL(SITE_URL),
  title: { default: "Clutter", template: "%s | Clutter" },
  description: "Draw your infrastructure once in building blocks, and deploy it to the clouds you link.",
  applicationName: "Clutter",
  robots: { index: false, follow: false },
};

// Every screen is dark, so the shadcn tokens use their dark values (the
// `dark` class turns on the `.dark` block in globals.css).
export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    // suppressHydrationWarning: browser extensions (QuillBot, password
    // managers) add attributes to <html> before React loads. It covers this
    // element's attributes only, not mismatches inside the page.
    <html lang="en" className={`dark ${geistSans.variable} ${geistMono.variable}`} suppressHydrationWarning>
      <body className="font-sans antialiased">
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
