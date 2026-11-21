"use client";

import { useEffect, useRef, type PointerEvent, type ReactNode } from "react";

type Props = {
  // The list or box that holds the cards.
  as?: "div" | "ul";
  className?: string;
  children: ReactNode;
};

// SpotlightGroup brightens the border of each card inside it that carries
// the `spotlight` class and a data-spotlight attribute, near the pointer
// (globals.css owns the look and turns it off for touch and reduced motion).
// Only a mouse or a pen moves it, at most once a frame; the cards look and
// work the same without it.
export default function SpotlightGroup({ as: Tag = "div", className, children }: Props) {
  const root = useRef<HTMLElement | null>(null);
  const frame = useRef(0);

  useEffect(() => () => cancelAnimationFrame(frame.current), []);

  const move = (e: PointerEvent<HTMLElement>) => {
    if (e.pointerType === "touch") return;
    const { clientX, clientY } = e;
    cancelAnimationFrame(frame.current);
    frame.current = requestAnimationFrame(() => {
      const el = root.current;
      if (!el) return;
      el.style.setProperty("--spot-alpha", "1");
      for (const card of el.querySelectorAll<HTMLElement>("[data-spotlight]")) {
        const r = card.getBoundingClientRect();
        card.style.setProperty("--spot-x", `${Math.round(clientX - r.left)}px`);
        card.style.setProperty("--spot-y", `${Math.round(clientY - r.top)}px`);
      }
    });
  };

  const leave = () => {
    cancelAnimationFrame(frame.current);
    root.current?.style.setProperty("--spot-alpha", "0");
  };

  return (
    <Tag
      ref={(el: HTMLElement | null) => {
        root.current = el;
      }}
      className={className}
      onPointerMove={move}
      onPointerLeave={leave}>
      {children}
    </Tag>
  );
}
