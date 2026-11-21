"use client";

import { useEffect, useRef, type ReactNode } from "react";

// The hero's motion controller. It wraps the hero's content and its
// decorative canvas (HeroCanvas) and does two things the CSS cannot:
// - Pointer parallax: each [data-depth] element shifts away from the pointer
//   by up to its depth in px, eased toward the pointer's place once per
//   animation frame. Only with a mouse, from 1280px wide, and not under
//   reduced motion; the rAF loop stops as soon as the tiles settle.
// - Pause: while the hero is off screen it sets data-paused, which stops
//   every hero-* animation (globals.css), and ignores the pointer.
// It reads nothing from layout per frame: the pointer's place is taken from
// the window's size, so writing transforms never forces a reflow.

const PARALLAX_QUERY = "(hover: hover) and (pointer: fine) and (min-width: 1280px)";
const REDUCE_QUERY = "(prefers-reduced-motion: reduce)";
// Time constant of the easing toward the pointer, in ms.
const SMOOTHING_MS = 140;

export function HeroMotion({ decor, children }: { decor: ReactNode; children: ReactNode }) {
  const root = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = root.current;
    if (!el) return;
    const wide = window.matchMedia(PARALLAX_QUERY);
    const reduce = window.matchMedia(REDUCE_QUERY);
    const layers = Array.from(el.querySelectorAll<HTMLElement>("[data-depth]")).map((node) => ({
      node,
      depth: Number(node.dataset.depth) || 0,
    }));

    let visible = true;
    let frame = 0;
    let last = 0;
    const target = { x: 0, y: 0 };
    const current = { x: 0, y: 0 };

    const enabled = () => visible && wide.matches && !reduce.matches;

    const write = () => {
      for (const { node, depth } of layers) {
        node.style.transform = `translate3d(${(current.x * depth).toFixed(2)}px, ${(current.y * depth).toFixed(2)}px, 0)`;
      }
    };

    const tick = (now: number) => {
      const dt = last ? Math.min(64, now - last) : 16;
      last = now;
      const k = 1 - Math.exp(-dt / SMOOTHING_MS);
      current.x += (target.x - current.x) * k;
      current.y += (target.y - current.y) * k;
      write();
      if (Math.abs(target.x - current.x) + Math.abs(target.y - current.y) > 0.002) {
        frame = requestAnimationFrame(tick);
      } else {
        frame = 0;
        last = 0;
      }
    };
    const start = () => {
      if (!frame) frame = requestAnimationFrame(tick);
    };
    const stop = () => {
      cancelAnimationFrame(frame);
      frame = 0;
      last = 0;
    };

    // Back to rest: eased when parallax is on, at once when it turned off.
    const rest = () => {
      target.x = 0;
      target.y = 0;
      if (enabled()) {
        start();
        return;
      }
      stop();
      current.x = 0;
      current.y = 0;
      for (const { node } of layers) node.style.transform = "";
    };

    const onMove = (e: PointerEvent) => {
      if (e.pointerType !== "mouse" || !enabled()) return;
      target.x = -((e.clientX / window.innerWidth) * 2 - 1);
      target.y = -((e.clientY / window.innerHeight) * 2 - 1);
      start();
    };
    // relatedTarget is null when the pointer leaves the window.
    const onOut = (e: PointerEvent) => {
      if (!e.relatedTarget) rest();
    };

    const observer = new IntersectionObserver(([entry]) => {
      visible = entry.isIntersecting;
      el.toggleAttribute("data-paused", !visible);
      if (!visible) stop();
    });
    observer.observe(el);

    window.addEventListener("pointermove", onMove, { passive: true });
    document.addEventListener("pointerout", onOut);
    window.addEventListener("blur", rest);
    wide.addEventListener("change", rest);
    reduce.addEventListener("change", rest);
    return () => {
      stop();
      observer.disconnect();
      window.removeEventListener("pointermove", onMove);
      document.removeEventListener("pointerout", onOut);
      window.removeEventListener("blur", rest);
      wide.removeEventListener("change", rest);
      reduce.removeEventListener("change", rest);
    };
  }, []);

  return (
    <div ref={root} className="hero-motion">
      {decor}
      {children}
    </div>
  );
}
