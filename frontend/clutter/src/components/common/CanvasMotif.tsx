import { cn } from "@/lib/utils";
import { MousePointer2 } from "lucide-react";
import type { ReactNode } from "react";

// Clutter's visual language, shared by the editor canvas, the site and empty
// states: a faint dot grid, and a dashed selection box with corner handles
// around the thing in focus, with an optional cursor label.

// DotGrid fills its positioned parent. "radial" fades it out towards the
// edges; "top" is the quieter 24px grid that fades out down from the top
// (bg-dot-grid-masked), for a hero, an empty state or onboarding.
export function DotGrid({ className, fade = "radial" }: { className?: string; fade?: "radial" | "top" | "none" }) {
  return (
    <div
      aria-hidden="true"
      className={cn(
        "pointer-events-none absolute inset-0",
        fade === "top" ? "bg-dot-grid-masked" : "bg-dot-grid",
        fade === "radial" && "[mask-image:radial-gradient(ellipse_at_center,black_35%,transparent_75%)]",
        className,
      )}
    />
  );
}

const handle = "border-primary bg-background absolute size-2 rounded-[2px] border";

type SelectionBoxProps = {
  children: ReactNode;
  // A small label beside a cursor at the lower right, as if someone were
  // dragging the box ("you", "Draw it once"). Decorative.
  cursorLabel?: string;
  className?: string;
};

// SelectionBox draws a dashed box with four handles around its children. The
// box and the cursor are decorative; the children keep their own meaning.
export function SelectionBox({ children, cursorLabel, className }: SelectionBoxProps) {
  return (
    <span className={cn("relative inline-flex items-center justify-center p-3", className)}>
      <span aria-hidden="true" className="border-primary/70 pointer-events-none absolute inset-0 rounded-[3px] border border-dashed" />
      <span aria-hidden="true" className={cn(handle, "-top-1 -left-1")} />
      <span aria-hidden="true" className={cn(handle, "-top-1 -right-1")} />
      <span aria-hidden="true" className={cn(handle, "-bottom-1 -left-1")} />
      <span aria-hidden="true" className={cn(handle, "-right-1 -bottom-1")} />
      {children}
      {cursorLabel && (
        <span aria-hidden="true" className="pointer-events-none absolute top-full left-full flex -translate-x-1 -translate-y-1 items-start gap-0.5">
          <MousePointer2 className="fill-primary text-primary size-4 -rotate-6" />
          <span className="bg-primary text-primary-foreground mt-3 rounded-full px-2 py-0.5 text-[11px] font-medium tracking-normal whitespace-nowrap">
            {cursorLabel}
          </span>
        </span>
      )}
    </span>
  );
}
