import type { Metadata } from "next";
import ThemeScope from "@/components/common/ThemeScope";
import { Clouds } from "@/components/landing/Clouds";
import { DrawOnce } from "@/components/landing/DrawOnce";
import { Examples } from "@/components/landing/Examples";
import { Faq } from "@/components/landing/Faq";
import { FinalCta } from "@/components/landing/FinalCta";
import { Hero } from "@/components/landing/Hero";
import { LandingHeader } from "@/components/landing/LandingHeader";
import { ReadAndDeploy } from "@/components/landing/ReadAndDeploy";
import { RevealOnScroll } from "@/components/landing/RevealOnScroll";
import { SeeEachCloud } from "@/components/landing/SeeEachCloud";
import { SeveralClouds } from "@/components/landing/SeveralClouds";
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

// Dark hero and dark close, light sections between, whatever theme the
// visitor chose in the app. Every page-wide background carries the grain
// (grain); cards and media on it do not. Every claim on the page is true
// today; the sources are named beside each section's text.
export default function Home() {
  return (
    <ThemeScope theme="light" className="bg-background grain text-foreground flex min-h-screen flex-col">
      <RevealOnScroll />
      <LandingHeader overHero />
      <main id="main" className="flex-1">
        <Hero />
        <DrawOnce />
        <SeeEachCloud />
        <ReadAndDeploy />
        <SeveralClouds />
        <Trust />
        <Examples />
        <Clouds />
        <Faq />
        <FinalCta />
      </main>
      <SiteFooter />
    </ThemeScope>
  );
}
