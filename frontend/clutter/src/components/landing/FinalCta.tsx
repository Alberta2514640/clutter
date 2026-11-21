import AssembleInView from "@/components/brand/AssembleInView";
import { pyramid, PYRAMID_MAX } from "@/components/brand/iso";
import IsoStack from "@/components/brand/IsoStack";
import { DotGrid } from "@/components/common/CanvasMotif";
import ThemeScope from "@/components/common/ThemeScope";
import { Button } from "@/components/ui/button";
import Link from "next/link";
import motion from "./motion.module.css";

const CTA_BLOCKS = pyramid(PYRAMID_MAX).map((b, i, all) => ({ ...b, accent: i === all.length - 1 }));

// The closing call to action, dark like the hero, on the same dot grid.
export function FinalCta() {
  return (
    <ThemeScope theme="dark" className="bg-hero grain text-hero-foreground relative isolate overflow-hidden">
      <section aria-labelledby="cta-title" className="relative">
        <DotGrid />
        <div className={`mx-auto max-w-3xl px-4 py-24 text-center md:px-8 md:py-32 ${motion.reveal}`}>
          {/* Six outline blocks with the logo's on top, built as the section comes into view. */}
          <AssembleInView className="mb-10 flex justify-center">
            <IsoStack blocks={CTA_BLOCKS} accent="brand" size={40} paper="var(--hero)" assemble />
          </AssembleInView>
          <h2 id="cta-title" className="text-4xl leading-[1.05] font-bold tracking-tighter text-balance md:text-6xl">
            Draw your first diagram.
          </h2>
          <p className="text-muted-foreground mx-auto mt-6 max-w-xl text-lg leading-relaxed">
            Start from an example or a blank canvas, link a cloud, and read the Terraform before anything is built.
          </p>
          <Button asChild size="lg" className="mt-10 h-12 w-full px-7 text-base sm:w-auto">
            <Link href="/login">Sign in with Google</Link>
          </Button>
          <p className="text-muted-foreground mt-4 text-sm">New here? Signing in creates your account.</p>
        </div>
      </section>
    </ThemeScope>
  );
}
