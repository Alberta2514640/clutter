"use client";

import { useEffect, useRef, type ReactNode } from "react";

// AssembleInView holds an assembling IsoStack inside it until it scrolls into
// view, then lets it assemble, once. While it is off screen it sets
// data-paused, which pauses the stack's idle loop. Without script, under
// reduced motion, or when it is already in view, the blocks are simply there
// (or assemble at once). Decorative, like the stack it holds: hidden from
// assistive technology.
export default function AssembleInView({ children, className }: { children: ReactNode; className?: string }) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = ref.current;
    if (!el || typeof IntersectionObserver === "undefined") return;
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;

    const pause = new IntersectionObserver((entries) => {
      el.toggleAttribute("data-paused", !entries.at(-1)?.isIntersecting);
    });
    pause.observe(el);
    if (el.getBoundingClientRect().top < window.innerHeight) return () => pause.disconnect();

    el.dataset.assemble = "waiting";
    const reveal = new IntersectionObserver(
      (entries) => {
        if (!entries.some((e) => e.isIntersecting)) return;
        el.dataset.assemble = "go";
        reveal.disconnect();
      },
      { rootMargin: "0px 0px -10% 0px" },
    );
    reveal.observe(el);
    return () => {
      pause.disconnect();
      reveal.disconnect();
    };
  }, []);

  return (
    <div ref={ref} aria-hidden="true" className={className}>
      {children}
    </div>
  );
}
