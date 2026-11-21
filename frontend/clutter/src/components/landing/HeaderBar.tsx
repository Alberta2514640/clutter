"use client";

import ThemeScope from "@/components/common/ThemeScope";
import { cn } from "@/lib/utils";
import { useEffect, useState } from "react";

// The public header's surface, always a dark scope. Over the home page's
// hero it is clear at the top of the page, so header and hero read as one
// surface, and turns to a blurred bar with a hairline border once the page
// scrolls. It starts as that bar, so a page without script, or one opened
// part-way down, never shows clear text over the light sections. Elsewhere
// (privacy, terms) it is a solid bar.
export function HeaderBar({ overHero, children }: { overHero: boolean; children: React.ReactNode }) {
  const [atTop, setAtTop] = useState(false);

  useEffect(() => {
    if (!overHero) return;
    const update = () => setAtTop(window.scrollY < 8);
    update();
    window.addEventListener("scroll", update, { passive: true });
    return () => window.removeEventListener("scroll", update);
  }, [overHero]);

  return (
    <ThemeScope
      theme="dark"
      data-at-top={overHero && atTop ? "" : undefined}
      className={cn(
        "sticky top-0 z-50 border-b transition-[background-color,border-color,backdrop-filter]",
        !overHero
          ? "bg-hero border-border"
          : atTop
            ? "-mb-16 border-transparent bg-transparent"
            : "bg-hero border-border -mb-16",
      )}>
      {children}
    </ThemeScope>
  );
}
