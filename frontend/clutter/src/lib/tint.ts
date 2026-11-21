// Tints: a Radix step 3 fill with its step 11 icon or initial, for the small
// badges that mark a kind of block or a team. The colours are tokens in
// globals.css (--tint-*); this file is the one place that names their
// classes, so Tailwind sees every class as written.

export type Tint = "blue" | "violet" | "amber" | "grass" | "orange" | "yellow" | "cyan" | "pink";

// A badge: an icon tile (globals.css) with the fill, the step 6 border and
// the icon colour.
const BADGE: Record<Tint, string> = {
  blue: "icon-tile bg-tint-blue border-tint-blue-edge text-tint-blue-fg",
  violet: "icon-tile bg-tint-violet border-tint-violet-edge text-tint-violet-fg",
  amber: "icon-tile bg-tint-amber border-tint-amber-edge text-tint-amber-fg",
  grass: "icon-tile bg-tint-grass border-tint-grass-edge text-tint-grass-fg",
  orange: "icon-tile bg-tint-orange border-tint-orange-edge text-tint-orange-fg",
  yellow: "icon-tile bg-tint-yellow border-tint-yellow-edge text-tint-yellow-fg",
  cyan: "icon-tile bg-tint-cyan border-tint-cyan-edge text-tint-cyan-fg",
  pink: "icon-tile bg-tint-pink border-tint-pink-edge text-tint-pink-fg",
};

// A bare icon: the icon colour only.
const ICON: Record<Tint, string> = {
  blue: "text-tint-blue-fg",
  violet: "text-tint-violet-fg",
  amber: "text-tint-amber-fg",
  grass: "text-tint-grass-fg",
  orange: "text-tint-orange-fg",
  yellow: "text-tint-yellow-fg",
  cyan: "text-tint-cyan-fg",
  pink: "text-tint-pink-fg",
};

export const tintBadgeClass = (t: Tint) => BADGE[t];
export const tintIconClass = (t: Tint) => ICON[t];

// The tints whose step 11 passes 4.5:1 on its step 3 in light and dark, so
// an initial (text, not an icon) stays readable.
const INITIAL_TINTS: Tint[] = ["violet", "grass", "pink"];

// tintForId picks a stable tint from an id, for a team's initial badge.
export function tintForId(id: string): Tint {
  let h = 0;
  for (let i = 0; i < id.length; i++) h = (h * 31 + id.charCodeAt(i)) >>> 0;
  return INITIAL_TINTS[h % INITIAL_TINTS.length];
}
