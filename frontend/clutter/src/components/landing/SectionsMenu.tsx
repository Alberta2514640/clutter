"use client";

import { Menu, X } from "lucide-react";
import Link from "next/link";
import { useEffect, useId, useRef, useState } from "react";
import { FOCUS_RING } from "./site";

export type SectionLink = { href: string; label: string };

// The section links on a narrow screen, where the header has no room for
// them: a Menu button that opens them in a small panel below it. A link, a
// click outside or Escape closes it; Escape puts focus back on the button.
// It renders inside the header's dark scope, so it needs no portal.
export function SectionsMenu({ sections }: { sections: SectionLink[] }) {
  const [open, setOpen] = useState(false);
  const panelId = useId();
  const root = useRef<HTMLDivElement>(null);
  const button = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== "Escape") return;
      setOpen(false);
      button.current?.focus();
    };
    const onPointer = (e: PointerEvent) => {
      if (!root.current?.contains(e.target as Node)) setOpen(false);
    };
    window.addEventListener("keydown", onKey);
    document.addEventListener("pointerdown", onPointer);
    return () => {
      window.removeEventListener("keydown", onKey);
      document.removeEventListener("pointerdown", onPointer);
    };
  }, [open]);

  return (
    <div ref={root} className="relative md:hidden">
      <button
        ref={button}
        type="button"
        aria-expanded={open}
        aria-controls={panelId}
        onClick={() => setOpen((o) => !o)}
        className={`text-foreground hover:bg-accent inline-flex h-9 items-center gap-1.5 rounded-md px-2.5 text-sm font-medium transition-colors ${FOCUS_RING}`}>
        {open ? <X className="size-4" aria-hidden="true" /> : <Menu className="size-4" aria-hidden="true" />}
        Menu
      </button>
      {open && (
        <nav id={panelId} aria-label="Sections" className="surface-raised absolute top-full right-0 mt-2 w-48 rounded-xl border p-1.5 shadow-lg animate-fade-in">
          <ul className="text-sm">
            {sections.map((s) => (
              <li key={s.href}>
                <Link
                  href={s.href}
                  onClick={() => setOpen(false)}
                  className={`text-muted-foreground hover:text-foreground hover:bg-accent flex min-h-10 items-center rounded-md px-3 transition-colors ${FOCUS_RING}`}>
                  {s.label}
                </Link>
              </li>
            ))}
          </ul>
        </nav>
      )}
    </div>
  );
}
