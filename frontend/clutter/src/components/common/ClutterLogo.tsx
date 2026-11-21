import styles from "@/components/brand/brand.module.css";
import { LOGO_COLOURS } from "@/components/brand/logoColours";
import { cn } from "@/lib/utils";
import { useId, type CSSProperties } from "react";

type Props = {
  /** Height of the mark in pixels. */
  size?: number;
  /** Show the "Clutter" name beside the mark. */
  withName?: boolean;
  /**
   * How the blocks move: "loop" (the loader), "once" (they drop in and stay),
   * "lift" (the top block rises 2px while the nearest `.group` is hovered or
   * focused). None under reduced motion.
   */
  motion?: "loop" | "once" | "lift";
  className?: string;
};

const { top, left, right, line } = LOGO_COLOURS;

/**
 * Clutter's mark: three stacked building blocks. The same drawing is
 * `public/logos/clutter-mark.svg` and the favicon (`src/app/icon.svg`); its
 * colours are `components/brand/logoColours.ts`.
 */
export default function ClutterLogo({ size = 32, withName = false, motion, className = "" }: Props) {
  const id = useId().replace(/:/g, "");
  // 2px on screen, in the mark's 64-unit drawing.
  const lift = { "--logo-lift": `${(-2 * 64) / size}px` } as CSSProperties;
  return (
    <span className={`inline-flex items-center gap-2 ${className}`}>
      <svg width={size} height={size} viewBox="0 0 64 64" role="img" aria-label={withName ? undefined : "Clutter"} aria-hidden={withName ? true : undefined}>
        <defs>
          <linearGradient id={`${id}t`} x1="0" y1="0" x2="1" y2="1">
            <stop offset="0" stopColor={top[0]} />
            <stop offset="1" stopColor={top[1]} />
          </linearGradient>
          <linearGradient id={`${id}l`} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" stopColor={left[0]} />
            <stop offset="1" stopColor={left[1]} />
          </linearGradient>
          <linearGradient id={`${id}r`} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" stopColor={right[0]} />
            <stop offset="1" stopColor={right[1]} />
          </linearGradient>
        </defs>
        <g stroke={line} strokeWidth="1.2" strokeLinejoin="round" className={cn(motion && styles[motion])} style={motion === "lift" ? lift : undefined}>
          <g>
            <path d="M6 34 L19 26.5 L32 34 L19 41.5 Z" fill={`url(#${id}t)`} />
            <path d="M6 34 L19 41.5 L19 56 L6 48.5 Z" fill={`url(#${id}l)`} />
            <path d="M19 41.5 L32 34 L32 48.5 L19 56 Z" fill={`url(#${id}r)`} />
          </g>
          <g>
            <path d="M32 34 L45 26.5 L58 34 L45 41.5 Z" fill={`url(#${id}t)`} />
            <path d="M32 34 L45 41.5 L45 56 L32 48.5 Z" fill={`url(#${id}l)`} />
            <path d="M45 41.5 L58 34 L58 48.5 L45 56 Z" fill={`url(#${id}r)`} />
          </g>
          <g>
            <path d="M19 19 L32 11.5 L45 19 L32 26.5 Z" fill={`url(#${id}t)`} />
            <path d="M19 19 L32 26.5 L32 41 L19 33.5 Z" fill={`url(#${id}l)`} />
            <path d="M32 26.5 L45 19 L45 33.5 L32 41 Z" fill={`url(#${id}r)`} />
          </g>
        </g>
      </svg>
      {withName && <span className="text-foreground text-lg font-bold tracking-tight">Clutter</span>}
    </span>
  );
}
