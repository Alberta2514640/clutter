// The isometric grid of Clutter's logo (public/logos/clutter-mark.svg): one
// step along x moves 13 right and 7.5 down, one step along y moves 13 left
// and 7.5 down, and one block up is 15. A block at (x, y, z) fills the unit
// cube from that corner; its top, left (+y) and right (+x) faces show. The
// logo's lower blocks sit at (0, 1, 0) and (1, 0, 0) and its top block at
// (0.5, 0.5, 1); the tops match the logo file, and a block is 15 tall where
// the file draws 14.5, so stacked outlines meet on one line.
//
// Pure: no React, no DOM, so Vitest runs it without the "@/" alias.

const ISO = { dx: 13, dy: 7.5, dz: 15 } as const;

// The width of one block on the grid, in grid units.
const BLOCK_WIDTH = 2 * ISO.dx;

export type IsoVariant = "outline" | "solid" | "brand";

export type IsoBlockSpec = {
  x: number;
  y: number;
  z?: number;
  variant?: IsoVariant;
  // A block kind from the catalog ("vm", "bucket"), for the solid variant.
  kind?: string;
  // The block a stack's `accent` colours.
  accent?: boolean;
};

// Two block indexes a thin line joins, like an edge on the canvas.
export type IsoLink = readonly [number, number];

export type IsoLayout = { blocks: IsoBlockSpec[]; links?: readonly IsoLink[] };

type Point = readonly [number, number];

// project turns a grid point into a point in the drawing.
export function project(x: number, y: number, z: number): Point {
  return [(x - y) * ISO.dx, (x + y) * ISO.dy - z * ISO.dz];
}

const path = (points: Point[]) => `M${points.map(([px, py]) => `${round(px)} ${round(py)}`).join(" L")} Z`;

// Four decimals keep the paths short and still exact for the grid's halves.
const round = (n: number) => Math.round(n * 10000) / 10000;

// blockFaces is the three visible faces of a block, as SVG path data.
export function blockFaces(b: IsoBlockSpec): { top: string; left: string; right: string } {
  const { x, y } = b;
  const z = b.z ?? 0;
  return {
    top: path([project(x, y, z + 1), project(x + 1, y, z + 1), project(x + 1, y + 1, z + 1), project(x, y + 1, z + 1)]),
    left: path([project(x, y + 1, z + 1), project(x + 1, y + 1, z + 1), project(x + 1, y + 1, z), project(x, y + 1, z)]),
    right: path([project(x + 1, y + 1, z + 1), project(x + 1, y, z + 1), project(x + 1, y, z), project(x + 1, y + 1, z)]),
  };
}

// blockCentre is the middle of a block, where a link starts and ends.
export function blockCentre(b: IsoBlockSpec): Point {
  return project(b.x + 0.5, b.y + 0.5, (b.z ?? 0) + 0.5);
}

export type Bounds = { minX: number; minY: number; maxX: number; maxY: number };

// bounds is the box the blocks' outlines fill, in drawing units.
export function bounds(blocks: IsoBlockSpec[]): Bounds {
  if (blocks.length === 0) return { minX: 0, minY: 0, maxX: 0, maxY: 0 };
  let minX = Infinity;
  let minY = Infinity;
  let maxX = -Infinity;
  let maxY = -Infinity;
  for (const b of blocks) {
    const z = b.z ?? 0;
    minX = Math.min(minX, (b.x - b.y - 1) * ISO.dx);
    maxX = Math.max(maxX, (b.x - b.y + 1) * ISO.dx);
    minY = Math.min(minY, (b.x + b.y) * ISO.dy - (z + 1) * ISO.dz);
    maxY = Math.max(maxY, (b.x + b.y + 2) * ISO.dy - z * ISO.dz);
  }
  return { minX, minY, maxX, maxY };
}

// paintOrder lists block indexes back to front, so a nearer block covers a
// farther one: the viewer looks down from +x, +y, +z.
export function paintOrder(blocks: IsoBlockSpec[]): number[] {
  const depth = (b: IsoBlockSpec) => b.x + b.y + (b.z ?? 0);
  return blocks
    .map((_, i) => i)
    .sort((a, b) => depth(blocks[a]) - depth(blocks[b]) || (blocks[a].z ?? 0) - (blocks[b].z ?? 0) || blocks[a].x - blocks[b].x || a - b);
}

// idleBlock is the block a stack's idle loop moves: the one its layout marks
// accent, else the last one painted (the top front one). -1 when there are
// no blocks.
export function idleBlock(blocks: IsoBlockSpec[]): number {
  const marked = blocks.findIndex((b) => b.accent);
  if (marked >= 0) return marked;
  return paintOrder(blocks).at(-1) ?? -1;
}

// blockPixels is one block's width on screen for a stack drawn at `size`:
// rounded to an even number of pixels (see frame).
function blockPixels(size: number): number {
  return Math.max(2, 2 * Math.round(size / 2));
}

// pixelsToUnits turns a distance on screen into drawing units for a stack
// drawn at `size`, so a motion moves the same few pixels at every size.
export function pixelsToUnits(px: number, size: number): number {
  return round((px * BLOCK_WIDTH) / blockPixels(size));
}

// The SVG frame for a stack drawn at `size` pixels per block width.
export type Frame = { width: number; height: number; viewBox: string };

// Room around the drawing, in pixels: half a pixel more than the 1px line, so
// every vertical edge lands on a pixel's centre.
const PAD_PX = 1.5;

// frame sizes the SVG so one grid step (13 units) is a whole number of
// pixels: the block width is rounded to an even number of pixels. Vertical
// edges then sit on half pixels and a 1px line is crisp at 1x and 2x. The
// width is even too, so a centred stack does not land on a half pixel. A
// block whose x - y is not a whole number is off the grid's columns and
// gives up this crispness.
export function frame(blocks: IsoBlockSpec[], size: number): Frame {
  const scale = blockPixels(size) / BLOCK_WIDTH;
  const b = bounds(blocks);
  const drawn = Math.round((b.maxX - b.minX) * scale + 2 * PAD_PX);
  const width = drawn % 2 === 0 ? drawn : drawn + 1;
  const height = Math.ceil((b.maxY - b.minY) * scale + 2 * PAD_PX);
  const pad = PAD_PX / scale;
  const box = [b.minX - pad, b.minY - pad, width / scale, height / scale].map(round);
  return { width, height, viewBox: box.join(" ") };
}

// The named compositions. Each marks one block as its accent.
export const ISO_PRESETS = {
  // The logo: two blocks below, one on top.
  cluster: {
    blocks: [
      { x: 0, y: 1 },
      { x: 1, y: 0 },
      { x: 0.5, y: 0.5, z: 1, accent: true },
    ],
  },
  // Two blocks side by side.
  pair: {
    blocks: [
      { x: 0, y: 1 },
      { x: 1, y: 0, accent: true },
    ],
  },
  // A small diagram: three blocks joined by two lines.
  linked: {
    blocks: [
      { x: 0, y: 2 },
      { x: 0, y: 0 },
      { x: 2, y: 0, accent: true },
    ],
    links: [
      [0, 1],
      [1, 2],
    ],
  },
  // Steps that fall towards the viewer.
  staircase: {
    blocks: [
      { x: 0, y: 0 },
      { x: 0, y: 0, z: 1 },
      { x: 0, y: 0, z: 2, accent: true },
      { x: 1, y: 0 },
      { x: 1, y: 0, z: 1 },
      { x: 2, y: 0 },
    ],
  },
  // Three blocks stacked straight up.
  tower: {
    blocks: [
      { x: 0, y: 0 },
      { x: 0, y: 0, z: 1 },
      { x: 0, y: 0, z: 2, accent: true },
    ],
  },
  // A stack knocked over: the middle block sliding off to the right, the top
  // one fallen to the floor beside it. Each x - y stays whole, on the grid's
  // columns.
  toppled: {
    blocks: [
      { x: 0, y: 1 },
      { x: 0.5, y: 0.5, z: 1 },
      { x: 2, y: 1, accent: true },
    ],
  },
} as const satisfies Record<string, IsoLayout>;

export type IsoPresetName = keyof typeof ISO_PRESETS;

// The places pyramid() fills, in order: the logo's three for up to three
// blocks, else a row of three, two above it, one on top.
const SMALL: IsoBlockSpec[][] = [
  [],
  [{ x: 0, y: 0 }],
  [
    { x: 0, y: 1 },
    { x: 1, y: 0 },
  ],
  ISO_PRESETS.cluster.blocks.map((b: IsoBlockSpec) => ({ x: b.x, y: b.y, z: b.z ?? 0 })),
];
const LARGE: IsoBlockSpec[] = [
  { x: 0, y: 2 },
  { x: 1, y: 1 },
  { x: 2, y: 0 },
  { x: 0.5, y: 1.5, z: 1 },
  { x: 1.5, y: 0.5, z: 1 },
  { x: 1, y: 1, z: 2 },
];

// The most blocks pyramid() draws.
export const PYRAMID_MAX = LARGE.length;

// pyramid places n blocks (at most PYRAMID_MAX) as the logo grows: one, two
// side by side, the logo's three, then a row of three with more on top.
export function pyramid(n: number): IsoBlockSpec[] {
  const count = Math.max(0, Math.min(PYRAMID_MAX, Math.floor(n)));
  const places = count < SMALL.length ? SMALL[count] : LARGE.slice(0, count);
  return places.map((p) => ({ ...p }));
}
