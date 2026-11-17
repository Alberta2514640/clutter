import type { Metadata } from "next";
import { Clouds } from "@/components/landing/Clouds";
import { Demo } from "@/components/landing/Demo";
import { Faq } from "@/components/landing/Faq";
import { FinalCta } from "@/components/landing/FinalCta";
import { Hero } from "@/components/landing/Hero";
import { HowItWorks } from "@/components/landing/HowItWorks";
import { LandingHeader } from "@/components/landing/LandingHeader";
import { SiteFooter } from "@/components/landing/SiteFooter";
import { Trust } from "@/components/landing/Trust";
import { HOME_DESCRIPTION, HOME_TITLE } from "@/components/landing/site";

// The one public page that search engines index (with /privacy and /terms).
// The Open Graph image is src/app/opengraph-image.png.
export const metadata: Metadata = {
  title: { absolute: HOME_TITLE },
  description: HOME_DESCRIPTION,
  alternates: { canonical: "/" },
  robots: { index: true, follow: true },
  openGraph: {
    type: "website",
    url: "/",
    siteName: "Clutter",
    locale: "en_CA",
    title: HOME_TITLE,
    description: HOME_DESCRIPTION,
  },
  twitter: {
    card: "summary_large_image",
    title: HOME_TITLE,
    description: HOME_DESCRIPTION,
  },
};

export default function Home() {
  return (
    <div className="flex min-h-screen flex-col bg-black text-white">
      <LandingHeader />
      <main id="main" className="flex-1">
        <Hero />
        <Demo />
        <HowItWorks />
        <Clouds />
        <Trust />
        <Faq />
        <FinalCta />
      </main>
      <SiteFooter />
    </div>
  );
}
