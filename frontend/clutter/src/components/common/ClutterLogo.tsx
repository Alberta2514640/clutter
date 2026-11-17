import { useId } from "react";

type Props = {
  /** Height of the mark in pixels. */
  size?: number;
  /** Show the "Clutter" name beside the mark. */
  withName?: boolean;
  className?: string;
};

/**
 * Clutter's mark: three stacked building blocks. The same drawing is
 * `public/logos/clutter-mark.svg` and the favicon (`src/app/icon.svg`).
 */
export default function ClutterLogo({ size = 32, withName = false, className = "" }: Props) {
  const id = useId().replace(/:/g, "");
  return (
    <span className={`inline-flex items-center gap-2 ${className}`}>
      <svg width={size} height={size} viewBox="0 0 64 64" role="img" aria-label={withName ? undefined : "Clutter"} aria-hidden={withName ? true : undefined}>
        <defs>
          <linearGradient id={`${id}t`} x1="0" y1="0" x2="1" y2="1">
            <stop offset="0" stopColor="#67e8f9" />
            <stop offset="1" stopColor="#5eead4" />
          </linearGradient>
          <linearGradient id={`${id}l`} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" stopColor="#3b82f6" />
            <stop offset="1" stopColor="#2563eb" />
          </linearGradient>
          <linearGradient id={`${id}r`} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" stopColor="#14b8a6" />
            <stop offset="1" stopColor="#0d9488" />
          </linearGradient>
        </defs>
        <g stroke="#0b1220" strokeWidth="1.2" strokeLinejoin="round">
          <path d="M6 34 L19 26.5 L32 34 L19 41.5 Z" fill={`url(#${id}t)`} />
          <path d="M6 34 L19 41.5 L19 56 L6 48.5 Z" fill={`url(#${id}l)`} />
          <path d="M19 41.5 L32 34 L32 48.5 L19 56 Z" fill={`url(#${id}r)`} />
          <path d="M32 34 L45 26.5 L58 34 L45 41.5 Z" fill={`url(#${id}t)`} />
          <path d="M32 34 L45 41.5 L45 56 L32 48.5 Z" fill={`url(#${id}l)`} />
          <path d="M45 41.5 L58 34 L58 48.5 L45 56 Z" fill={`url(#${id}r)`} />
          <path d="M19 19 L32 11.5 L45 19 L32 26.5 Z" fill={`url(#${id}t)`} />
          <path d="M19 19 L32 26.5 L32 41 L19 33.5 Z" fill={`url(#${id}l)`} />
          <path d="M32 26.5 L45 19 L45 33.5 L32 41 Z" fill={`url(#${id}r)`} />
        </g>
      </svg>
      {withName && <span className="text-lg font-bold tracking-tight text-slate-100">Clutter</span>}
    </span>
  );
}
