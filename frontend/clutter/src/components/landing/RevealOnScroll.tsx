"use client";

import { useEffect } from "react";
import motion from "./motion.module.css";

// Arms the home page's section reveal (motion.module.css). It waits for the
// visitor's first scroll, then hides only the .reveal elements still wholly
// below the window and shows each one, once, as it comes into view. Without
// script, under reduced motion, or before any scroll, every section is simply
// there. Renders nothing.
export function RevealOnScroll() {
  useEffect(() => {
    if (typeof IntersectionObserver === "undefined") return;
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;

    const io = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (!entry.isIntersecting) continue;
          (entry.target as HTMLElement).dataset.reveal = "shown";
          io.unobserve(entry.target);
        }
      },
      { rootMargin: "0px 0px -8% 0px" },
    );

    const arm = () => {
      window.removeEventListener("scroll", arm);
      for (const el of document.querySelectorAll<HTMLElement>(`.${motion.reveal}`)) {
        if (el.getBoundingClientRect().top < window.innerHeight) continue;
        el.dataset.reveal = "waiting";
        io.observe(el);
      }
    };

    window.addEventListener("scroll", arm, { passive: true });
    return () => {
      window.removeEventListener("scroll", arm);
      io.disconnect();
    };
  }, []);

  return null;
}
