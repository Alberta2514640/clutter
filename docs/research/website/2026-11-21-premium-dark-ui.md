# Premium dark UI: colour, type and surface values

Read 2026-11-21 for the website's palette after the person found the first dark theme tinted and glowy (ADR-028). Sources are official pages; values are only those the sources publish.

## What the references share

Linear (<https://linear.app>), Vercel (<https://vercel.com/geist>), Resend (<https://resend.com>) and Raycast (<https://www.raycast.com>), seen live on 2026-11-21: near-black backgrounds, neutral greys, white primary buttons on dark, colour kept for status, small dense type inside apps, and borders rather than shadows to separate surfaces.

## Sources with values

- Radix Colors 3.0.0 publishes every step's hex in its CSS files (<https://unpkg.com/@radix-ui/colors@3.0.0/gray-dark.css> and siblings) and the role of each step (<https://www.radix-ui.com/colors/docs/palette-composition/understanding-the-scale>): 1 app background, 2 subtle background, 3 to 5 component default, hover and selected, 6 non-interactive border, 7 interactive border, 8 strong border and focus, 9 and 10 solid and hovered solid, 11 low-contrast text, 12 high-contrast text.
- Radix gray dark, steps 1 to 12: #111111, #191919, #222222, #2a2a2a, #313131, #3a3a3a, #484848, #606060, #6e6e6e, #7b7b7b, #b4b4b4, #eeeeee.
- Radix recommends gray text in apps and warns that a saturated gray can clash with colourful components (<https://www.radix-ui.com/colors/docs/palette-composition/composing-a-palette>).

## Sources with roles but no values

- Vercel Geist (<https://vercel.com/geist/colors>, <https://vercel.com/geist/typography>, <https://vercel.com/geist/materials>): gray steps 100 to 300 for backgrounds, 400 to 600 for borders, 900 and 1000 for text; radius 6 for small surfaces and 12 for large ones, modals and menus; type roles heading, label, copy and button, with copy-14 the most used and mono only for code, paths, ids and timestamps. The pages give no hex or pixel values.
- Vercel's web interface guidelines (<https://vercel.com/design/guidelines>): two-layer shadows and a semi-transparent border for floating layers only, hover more contrasting than rest, `:focus-visible` rings, inputs at least 16px on phones, hit targets at least 24px (44px on mobile), tabular numbers in tables, every state designed (empty, sparse, dense, error) with a next step, animate only transform and opacity, honour reduced motion.
- Linear's redesign notes (<https://linear.app/now/how-we-redesigned-the-linear-ui>, <https://linear.app/now/behind-the-latest-design-refresh>): neutral greys generated in LCH, a quieter sidebar so content wins, fewer and smaller icons without coloured backgrounds, "structure should be felt, not seen".

## Not found

Raw Geist values (the live page copies them on right click only), Railway, Resend and Raycast colour values (no official brand values; third-party ones conflict), Supabase values (token names only).
