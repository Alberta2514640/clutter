// The logo's colours: each face's gradient stops (top, left, right) and the
// line, as public/logos/clutter-mark.svg and src/app/icon.svg draw them.
// ClutterLogo draws the gradients; the brand block (IsoStack) is flat, in
// each face's first stop. Change the two SVG files with this.
export const LOGO_COLOURS = {
  top: ["#67e8f9", "#5eead4"],
  left: ["#3b82f6", "#2563eb"],
  right: ["#14b8a6", "#0d9488"],
  line: "#0b1220",
} as const;
