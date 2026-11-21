"use client";

import { useId } from "react";

// HelpTip is the small "?" that holds an explanation: it shows on hover or
// keyboard focus, and screen readers read it as the button's description.
// align picks the side the card grows from, so it stays inside a panel;
// "row" spans the card across the nearest positioned parent (a field's label
// row), so it fits whatever side of the panel the "?" sits on.
// The button is a 24 px target (WCAG 2.5.8) around a 16 px circle; its
// negative margin keeps the 16 px footprint, so rows do not grow.
export default function HelpTip({ text, label, align = "start" }: { text: string; label: string; align?: "start" | "end" | "row" }) {
  const id = useId();
  const place = align === "row" ? "left-0 right-0 w-auto" : align === "end" ? "right-0 w-60" : "left-0 w-60";
  return (
    <span className={`group/tip inline-flex shrink-0 align-middle ${align === "row" ? "" : "relative"}`}>
      <button
        type="button"
        aria-label={label}
        aria-describedby={id}
        onClick={(e) => e.preventDefault()}
        className="nodrag nopan group/help -m-1 inline-flex size-6 cursor-help items-center justify-center outline-none">
        <span
          aria-hidden
          className="text-muted-foreground group-hover/help:border-link group-hover/help:text-link group-focus-visible/help:ring-ring/50 border-border inline-flex size-4 items-center justify-center rounded-full border text-[10px] leading-none font-semibold group-focus-visible/help:ring-[3px]">
          ?
        </span>
      </button>
      <span
        role="tooltip"
        id={id}
        className={`surface-raised text-popover-foreground pointer-events-none invisible absolute top-6 z-50 rounded-lg border px-3 py-2 text-left text-xs leading-snug font-normal tracking-normal normal-case opacity-0 transition-opacity group-focus-within/tip:visible group-focus-within/tip:opacity-100 group-hover/tip:visible group-hover/tip:opacity-100 ${place}`}>
        {text}
      </span>
    </span>
  );
}
