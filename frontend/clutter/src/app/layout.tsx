// app/layout.tsx

import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import { themeScript } from "@/components/common/theme";
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

// The server renders the default theme, dark. The inline script in <head>
// runs before the first paint and takes `dark` off <html> when the viewer
// chose light (or system on a light system), so a reload never flashes the
// wrong theme
// (components/common/theme.ts). Parts with a fixed theme are ThemeScopes.
export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    // suppressHydrationWarning: the theme script changes this element's class
    // and style before React loads, and browser extensions (QuillBot, password
    // managers) add attributes to it. It covers this element only.
    <html lang="en" className={`dark ${geistSans.variable} ${geistMono.variable}`} style={{ colorScheme: "dark" }} suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeScript }} />
      </head>
      <body className="font-sans antialiased">
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
