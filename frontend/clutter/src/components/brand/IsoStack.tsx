import { KIND_TINT } from "@/lib/features/catalog/display";
import { cn } from "@/lib/utils";
import type { CSSProperties } from "react";
import styles from "./brand.module.css";
import { blockCentre, blockFaces, frame, idleBlock, ISO_PRESETS, paintOrder, pixelsToUnits, type IsoBlockSpec, type IsoLink, type IsoPresetName, type IsoVariant } from "./iso";
import { LOGO_COLOURS } from "./logoColours";

// What a stack's accent block is: the logo's colours, a block kind's colours
// ("vm", "bucket"), or an outline like the rest.
export type IsoAccent = "brand" | "outline" | (string & {});

// How a stack's idle block moves: "lift" rises and settles, "rock" tips on
// its left edge and settles.
export type IsoIdle = "lift" | "rock";

// How far a lifting block rises, in pixels on screen at every size.
const IDLE_LIFT_PX = 2.5;

type Layout = { preset: IsoPresetName; blocks?: never; links?: never } | { preset?: never; blocks: IsoBlockSpec[]; links?: readonly IsoLink[] };

export type IsoStackProps = Layout & {
  // Colours the block a layout marks `accent`. Default: no accent.
  accent?: IsoAccent;
  // Pixel width of one block (rounded to an even number). Default 24.
  size?: number;
  // The colour behind the stack, which outline faces are filled with so a
  // nearer block hides the lines behind it. Default --iso-paper from a
  // parent, else the page background.
  paper?: string;
  // Drop the blocks in one by one when shown (or when an AssembleInView
  // around it comes into view).
  assemble?: boolean;
  // Once in place, one block (the layout's accent, else the top front one)
  // moves on a slow loop. Inside an AssembleInView it pauses off screen;
  // under reduced motion it is still. Default: no idle loop.
  idle?: IsoIdle;
  className?: string;
};

type Vars = CSSProperties & Record<`--${string}`, string | number>;

// The colour behind a stack: its `paper`, else --iso-paper from a parent (an
// EmptyState sets the card's), else the page background.
const PAPER = "var(--iso-paper, var(--background))";

// The outline's line: gray 7 (--input) a quarter of the way to gray 11
// (--muted-foreground), about gray 8 in both themes. Gray 7 alone fades on
// white.
const OUTLINE ="color-mix(in oklab, var(--input) 75%, var(--muted-foreground))";

// faceVars is the fill of each face and the line for one block. Outline faces
// are the paper with a faint shade; solid faces are the kind's Radix steps 3
// and 6 and a step-9-like mix of 11 into 6; brand faces are the logo's.
function faceVars(variant: IsoVariant, kind: string | undefined): Vars {
  if (variant === "brand") {
    return { "--iso-top": LOGO_COLOURS.top[0], "--iso-left": LOGO_COLOURS.left[0], "--iso-right": LOGO_COLOURS.right[0], "--iso-line": LOGO_COLOURS.line };
  }
  const tint = variant === "solid" && kind ? KIND_TINT[kind] : undefined;
  if (variant === "solid") {
    const [step3, step6, step11] = tint
      ? [`var(--tint-${tint})`, `var(--tint-${tint}-edge)`, `var(--tint-${tint}-fg)`]
      : ["var(--muted)", "var(--border)", "var(--muted-foreground)"];
    return {
      "--iso-top": step3,
      "--iso-left": step6,
      "--iso-right": `color-mix(in oklab, ${step11} 70%, ${step6})`,
      "--iso-line": `color-mix(in oklab, ${step11} 45%, ${step6})`,
    };
  }
  return {
    "--iso-top": `color-mix(in oklab, var(--foreground) 5%, ${PAPER})`,
    "--iso-left": PAPER,
    "--iso-right": `color-mix(in oklab, var(--foreground) 2.5%, ${PAPER})`,
    "--iso-line": OUTLINE,
  };
}

function resolve(b: IsoBlockSpec, accent: IsoAccent | undefined): { variant: IsoVariant; kind?: string } {
  if (b.accent && accent) {
    if (accent === "brand") return { variant: "brand" };
    if (accent === "outline") return { variant: "outline" };
    return { variant: "solid", kind: accent };
  }
  return { variant: b.variant ?? "outline", kind: b.kind };
}

// IsoStack draws blocks on the logo's isometric grid, from a named preset or
// a layout. It is decorative: hidden from assistive technology.
export default function IsoStack({ preset, blocks: own, links: ownLinks, accent, size = 24, paper, assemble = false, idle, className }: IsoStackProps) {
  const layout = preset ? ISO_PRESETS[preset] : { blocks: own ?? [], links: ownLinks };
  const blocks: IsoBlockSpec[] = [...layout.blocks];
  const links = ("links" in layout ? layout.links : undefined) ?? [];
  const { width, height, viewBox } = frame(blocks, size);
  const order = paintOrder(blocks);
  const idleIndex = idle ? idleBlock(blocks) : -1;
  // The idle loop waits for every block to land (--iso-n) and lifts the same
  // pixels at every size (--iso-lift, in drawing units).
  const idleVars: Vars = { "--iso-n": blocks.length, "--iso-lift": `${-pixelsToUnits(IDLE_LIFT_PX, size)}px` };

  return (
    <svg
      aria-hidden="true"
      focusable="false"
      width={width}
      height={height}
      viewBox={viewBox}
      className={cn(styles.stack, assemble && styles.assemble, "shrink-0", className)}
      style={paper ? ({ "--iso-paper": paper } as Vars) : undefined}>
      {links.map(([a, b]) => {
        if (!blocks[a] || !blocks[b]) return null;
        const [x1, y1] = blockCentre(blocks[a]);
        const [x2, y2] = blockCentre(blocks[b]);
        return <line key={`${a}-${b}`} className={styles.link} style={{ stroke: OUTLINE }}x1={x1} y1={y1} x2={x2} y2={y2} />;
      })}
      {order.map((index, step) => {
        const b = blocks[index];
        const { variant, kind } = resolve(b, accent);
        const faces = blockFaces(b);
        const paths = (
          <>
            <path className={cn(styles.face, styles.top)} d={faces.top} />
            <path className={cn(styles.face, styles.left)} d={faces.left} />
            <path className={cn(styles.face, styles.right)} d={faces.right} />
          </>
        );
        // The idle loop runs on an inner group, so it never fights the drop.
        return (
          <g key={index} className={styles.block} style={{ ...faceVars(variant, kind), "--i": step } as Vars}>
            {index === idleIndex ? (
              <g className={idle === "rock" ? styles.idleRock : styles.idleLift} style={idleVars}>
                {paths}
              </g>
            ) : (
              paths
            )}
          </g>
        );
      })}
    </svg>
  );
}
