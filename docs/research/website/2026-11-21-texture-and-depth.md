# Texture and depth without gradients or glows

Read 2026-11-21, after the person found the neutral theme's surfaces flat. Official sources where they exist; the opacity numbers are community starting points to check by eye. No primary source was found showing how Linear, Vercel, Stripe, Raycast or Resend apply grain or spotlights.

## Techniques

| Technique | Values to start from | Source |
| --- | --- | --- |
| Elevation by lighter surface in dark | Radix steps 1 and 2 for the page, 3 for cards, 4 hover, 5 selected | <https://www.radix-ui.com/colors/docs/palette-composition/understanding-the-scale>, <https://m2.material.io/design/color/dark-theme> |
| Hairline borders from the scale | step 6 for cards, 7 for controls, 8 for focus; fewer dividers | Radix, above; <https://linear.app/now/behind-the-latest-design-refresh> (2026-03-12) |
| Top inner highlight | `inset 0 1px 0` white at about 0.06 in dark | <https://tailwindcss.com/docs/box-shadow> for the utility; value derived |
| Layered tinted shadows, mainly in light | 3 to 5 faint layers, doubling offset and blur, one light source | <https://www.joshwcomeau.com/css/designing-shadows/> |
| A small set of materials | base and small surfaces at radius 6, larger ones at 12, floating layers with more lift | <https://vercel.com/geist/materials> |
| Static grain on one page layer | SVG `feTurbulence` fractal noise, base frequency about 0.9, 3 octaves, opacity about 0.025 light and 0.035 dark | <https://css-tricks.com/grainy-gradients/>, <https://blog.master.dev/grainy-gradients/> (2025-06-13) |
| Masked dot or line grid | 1px at step 4 to 5, 24px pitch, faded with `mask-image` | <https://ishadeed.com/article/css-masking/> |
| Pointer spotlight on card borders | radial mask from the pointer position, off under reduced motion, content usable without it | <https://developer.chrome.com/docs/modern-web-guidance/guides/ui-behaviors/interactive-content-reveal> |
| Hierarchy by scale, spacing and weight rather than effects | one leading element per view | <https://world.hey.com/kostac/spot-the-slop-a-ui-designer-s-guide-to-fixing-ai-defaults-4c448c9c> (2026-07-23) |

## Read as cheap

Purple-to-blue gradients, glass panels everywhere, decorative glows that signal nothing, the same radius and padding on every card, coloured left-edge lines on panels, and heavy backdrop blur (the practitioner piece above, and the articles it summarises).

## Not found

Vendor-published shadow or overlay numbers (Geist's materials page has none), and any first-hand account of how the named brands apply grain.
