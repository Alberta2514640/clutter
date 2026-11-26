"use client";

import AssembleInView from "@/components/brand/AssembleInView";
import IsoStack from "@/components/brand/IsoStack";
import { DotGrid } from "@/components/common/CanvasMotif";
import CloudLogo from "@/components/common/CloudLogo";
import ThemeScope from "@/components/common/ThemeScope";
import { SHOTS, SIGNIN_SHOTS } from "@/components/landing/shots";
import { cn } from "@/lib/utils";
import Image from "next/image";
import { useEffect, useState } from "react";

// The sign-in page's right half: what Clutter does, in three real
// screenshots. It auto-advances about every 6 seconds, pauses while the
// pointer or focus is on it, and never moves by itself under reduced motion.

const SLIDES = [
  { id: "design", title: "Design", line: "Draw your setup in plain building blocks: servers, networks, storage.", shot: SHOTS.editor },
  { id: "connect", title: "Connect a cloud", line: "Connect your own AWS or OpenStack account to build in.", shot: SIGNIN_SHOTS.connect },
  { id: "deploy", title: "Deploy", line: "Preview the Terraform, then build it in your own account.", shot: SIGNIN_SHOTS.deploy },
] as const;

const ADVANCE_MS = 6000;

function usePrefersReducedMotion() {
  const [reduced, setReduced] = useState(true);
  useEffect(() => {
    const query = window.matchMedia("(prefers-reduced-motion: reduce)");
    const update = () => setReduced(query.matches);
    update();
    query.addEventListener("change", update);
    return () => query.removeEventListener("change", update);
  }, []);
  return reduced;
}

export default function BrandPanel() {
  const [index, setIndex] = useState(0);
  const [hovered, setHovered] = useState(false);
  const [focused, setFocused] = useState(false);
  const reduced = usePrefersReducedMotion();
  const paused = hovered || focused || reduced;

  useEffect(() => {
    if (paused) return;
    const timer = window.setTimeout(() => setIndex((i) => (i + 1) % SLIDES.length), ADVANCE_MS);
    return () => window.clearTimeout(timer);
  }, [index, paused]);

  const current = SLIDES[index];

  return (
    <ThemeScope
      theme="dark"
      className="bg-hero text-hero-foreground relative isolate hidden overflow-hidden lg:flex lg:flex-col lg:justify-center">
      <DotGrid className="-z-10 opacity-70" />
      {/* Tucked into the corner, clear of the slide controls; only where the margin beside the content is wide enough. */}
      <AssembleInView className="pointer-events-none absolute -right-3 -bottom-3 -z-10 hidden opacity-50 xl:flex">
        <IsoStack preset="staircase" size={40} paper="var(--hero)" assemble idle="lift" />
      </AssembleInView>

      <section
        aria-roledescription="carousel"
        aria-label="What Clutter does"
        onMouseEnter={() => setHovered(true)}
        onMouseLeave={() => setHovered(false)}
        onFocus={() => setFocused(true)}
        onBlur={(event) => {
          if (!event.currentTarget.contains(event.relatedTarget as Node | null)) setFocused(false);
        }}
        className="mx-auto w-full max-w-xl px-10 py-16 xl:px-0">
        <div className="relative aspect-[16/10] overflow-hidden rounded-xl border border-hero-foreground/10 bg-hero-foreground/5 shadow-2xl shadow-black/40">
          <div aria-hidden="true" className="flex h-6 items-center gap-1.5 border-b border-hero-foreground/10 bg-hero-foreground/5 px-3">
            <span className="size-2 rounded-full bg-hero-foreground/20" />
            <span className="size-2 rounded-full bg-hero-foreground/20" />
            <span className="size-2 rounded-full bg-hero-foreground/20" />
          </div>
          <div className="relative h-[calc(100%-1.5rem)]">
            {SLIDES.map((slide, i) => (
              <div
                key={slide.id}
                role="group"
                aria-roledescription="slide"
                aria-label={`${i + 1} of ${SLIDES.length}: ${slide.title}`}
                aria-hidden={i !== index}
                className={cn(
                  "motion-slow ease-standard absolute inset-0 transition-opacity",
                  i === index ? "opacity-100" : "opacity-0",
                )}>
                <Image
                  src={slide.shot.src}
                  alt={slide.shot.alt}
                  fill
                  sizes="576px"
                  priority={i === 0}
                  className="object-cover object-top"
                />
              </div>
            ))}
          </div>
        </div>

        <h2 className="mt-10 text-3xl font-semibold tracking-tight text-balance xl:text-4xl">
          Draw it once. Build it on AWS, OpenStack, or both.
        </h2>
        <p className="mt-3 max-w-md text-base text-hero-foreground/65">
          Clutter turns a diagram into Terraform and runs it in a cloud account you own.
        </p>
        <p className="mt-5 flex items-center gap-4 text-sm text-hero-foreground/70">
          <span className="inline-flex items-center gap-2">
            <CloudLogo connector="aws" size={20} />
            AWS
          </span>
          <span className="inline-flex items-center gap-2">
            <CloudLogo connector="general-openstack" size={20} />
            OpenStack
          </span>
        </p>

        <div className="mt-10 flex items-center justify-between gap-6 border-t border-hero-foreground/10 pt-6">
          <p aria-live={paused ? "polite" : "off"} className="min-h-10 text-sm text-hero-foreground/80">
            <span className="text-hero-foreground font-medium">{current.title}.</span> {current.line}
          </p>
          <div className="flex shrink-0 items-center gap-1">
            {SLIDES.map((slide, i) => (
              <button
                key={slide.id}
                type="button"
                onClick={() => setIndex(i)}
                aria-label={`Show slide ${i + 1}: ${slide.title}`}
                aria-current={i === index ? "true" : undefined}
                className="group flex size-6 items-center justify-center rounded-full outline-none focus-visible:ring-[3px] focus-visible:ring-hero-foreground/40">
                <span
                  className={cn(
                    "motion-base ease-standard block h-1.5 w-5 rounded-full transition-[scale,background-color]",
                    i === index ? "bg-primary scale-x-100" : "scale-x-[0.3] bg-hero-foreground/30 group-hover:bg-hero-foreground/60",
                  )}
                />
              </button>
            ))}
          </div>
        </div>
      </section>
    </ThemeScope>
  );
}
