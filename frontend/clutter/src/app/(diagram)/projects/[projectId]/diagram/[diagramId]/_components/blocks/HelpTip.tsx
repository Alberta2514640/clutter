"use client";

import { useId } from "react";

// HelpTip is the small "?" that holds an explanation: it shows on hover or
// keyboard focus, and screen readers read it as the button's description.
// align picks the side the card grows from, so it stays inside a panel;
// "row" spans the card across the nearest positioned parent (a field's label
// row), so it fits whatever side of the panel the "?" sits on.
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
        className="nodrag nopan inline-flex h-4 w-4 cursor-help items-center justify-center rounded-full border border-slate-600 text-[10px] font-semibold leading-none text-slate-400 hover:border-teal-400 hover:text-teal-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-teal-300">
        ?
      </button>
      <span
        role="tooltip"
        id={id}
        className={`pointer-events-none invisible absolute top-6 z-50 rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-left text-xs font-normal normal-case leading-snug tracking-normal text-slate-200 opacity-0 shadow-xl transition-opacity group-focus-within/tip:visible group-focus-within/tip:opacity-100 group-hover/tip:visible group-hover/tip:opacity-100 ${place}`}>
        {text}
      </span>
    </span>
  );
}
