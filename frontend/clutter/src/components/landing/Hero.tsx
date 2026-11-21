import { DotGrid, SelectionBox } from "@/components/common/CanvasMotif";
import ThemeScope from "@/components/common/ThemeScope";
import { Button } from "@/components/ui/button";
import { Play } from "lucide-react";
import Link from "next/link";
import { HeroCanvas } from "./HeroCanvas";
import { HeroMotion } from "./HeroMotion";
import { Screenshot } from "./Screenshot";
import { SHOTS } from "./shots";

// The hero: one headline on the canvas motif with a small living diagram
// beside it (HeroCanvas, moved by HeroMotion), the sign-in action, and the
// real product shot in a frame that straddles the dark hero and the light
// page below.
export function Hero() {
  return (
    <>
      {/* pt-16: the header (LandingHeader overHero) lies over this top strip. */}
      <ThemeScope theme="dark" className="bg-hero grain text-hero-foreground relative isolate overflow-hidden pt-16">
        <section aria-labelledby="hero-title">
          <DotGrid fade="top" />

          <HeroMotion decor={<HeroCanvas />}>
            <div className="relative mx-auto max-w-6xl px-4 pt-20 pb-28 text-center md:px-8 md:pt-28 md:pb-36">
              <p className="micro-label mb-6">Diagram to Terraform to your cloud</p>
              <h1 id="hero-title" className="mx-auto max-w-5xl text-[2.75rem] leading-[1.02] font-bold tracking-tighter text-balance sm:text-7xl lg:text-[5.75rem]">
                One{" "}
                {/* hero-selection: the handles pulse and the cursor idles (globals.css). */}
                <SelectionBox cursorLabel="you" className="hero-selection mx-0.5 px-2 py-0.5 sm:px-3">
                  diagram.
                </SelectionBox>
                <br />
                Every cloud you link.
              </h1>
              <p className="text-muted-foreground mx-auto mt-8 max-w-2xl text-lg leading-relaxed text-pretty md:text-xl">
                Draw your system once in plain building blocks. Clutter writes the Terraform and builds it on AWS and OpenStack, inside the accounts you link.
              </p>
              <div className="mt-10 flex flex-col items-stretch justify-center gap-3 sm:flex-row sm:items-center">
                {/* The one monochrome main action: white on the dark hero. */}
                <Button asChild size="lg" className="bg-foreground text-background hover:bg-foreground/90 h-12 px-7 text-base">
                  <Link href="/login">Sign in with Google</Link>
                </Button>
                <Button asChild size="lg" variant="outline" className="h-12 px-7 text-base">
                  <Link href="#demo">
                    <Play aria-hidden="true" />
                    Watch the demo
                  </Link>
                </Button>
              </div>
              <p className="text-muted-foreground mt-4 text-sm">New here? Signing in creates your account.</p>
            </div>
          </HeroMotion>
        </section>
      </ThemeScope>

      {/* The shot's upper part sits on the hero, its lower part on the light page. */}
      <div className="relative px-4 md:px-8">
        <div aria-hidden="true" className="dark bg-hero grain absolute inset-x-0 top-0 h-1/2" />
        <div className="relative mx-auto -mt-16 max-w-6xl md:-mt-24">
          <Screenshot shot={SHOTS.editor} sizes="(min-width: 1216px) 1152px, calc(100vw - 32px)" hero />
        </div>
      </div>
    </>
  );
}
