import { describe, expect, it } from "vitest";
import { blockFaces, bounds, frame, idleBlock, ISO_PRESETS, paintOrder, pixelsToUnits, project, pyramid, PYRAMID_MAX } from "./iso";

// The logo's paths (public/logos/clutter-mark.svg), moved so its lower left
// block's top centre (19, 34) is this grid's (0, 1, 0) top centre (-13, 0).
const LOGO_OFFSET = [32, 34] as const;
const toLogo = (d: string) =>
  d.replace(/(-?[\d.]+) (-?[\d.]+)/g, (_, x: string, y: string) => `${Number(x) + LOGO_OFFSET[0]} ${Number(y) + LOGO_OFFSET[1]}`);

describe("project", () => {
  it("moves 13 across and 7.5 down a grid step, and 15 up a block", () => {
    expect(project(1, 0, 0)).toEqual([13, 7.5]);
    expect(project(0, 1, 0)).toEqual([-13, 7.5]);
    expect(project(0, 0, 1)).toEqual([0, -15]);
  });
});

describe("blockFaces", () => {
  it("draws the logo's top faces exactly", () => {
    const [lowerLeft, lowerRight, top] = ISO_PRESETS.cluster.blocks;
    expect(toLogo(blockFaces(lowerLeft).top)).toBe("M19 26.5 L32 34 L19 41.5 L6 34 Z");
    expect(toLogo(blockFaces(lowerRight).top)).toBe("M45 26.5 L58 34 L45 41.5 L32 34 Z");
    expect(toLogo(blockFaces(top).top)).toBe("M32 11.5 L45 19 L32 26.5 L19 19 Z");
  });

  it("draws a block 15 tall, on the grid's step", () => {
    expect(blockFaces({ x: 0, y: 0 }).left).toBe("M-13 -7.5 L0 0 L0 15 L-13 7.5 Z");
    expect(blockFaces({ x: 0, y: 0 }).right).toBe("M0 0 L13 -7.5 L13 7.5 L0 15 Z");
  });
});

describe("bounds", () => {
  it("is the outline of every block", () => {
    expect(bounds([{ x: 0, y: 0 }])).toEqual({ minX: -13, minY: -15, maxX: 13, maxY: 15 });
    expect(bounds(ISO_PRESETS.tower.blocks)).toEqual({ minX: -13, minY: -45, maxX: 13, maxY: 15 });
  });

  it("is empty for no blocks", () => {
    expect(bounds([])).toEqual({ minX: 0, minY: 0, maxX: 0, maxY: 0 });
  });
});

describe("paintOrder", () => {
  it("paints back to front and bottom to top", () => {
    expect(paintOrder(ISO_PRESETS.cluster.blocks)).toEqual([0, 1, 2]);
    expect(paintOrder([{ x: 0, y: 0, z: 2 }, { x: 1, y: 1 }, { x: 0, y: 0 }])).toEqual([2, 1, 0]);
  });

  it("paints a block before the ones in front of it and on top of it", () => {
    // (0, 0, 0) is behind (1, 0, 0) and under (0, 0, 1).
    const order = paintOrder([{ x: 1, y: 0 }, { x: 0, y: 0 }, { x: 0, y: 0, z: 1 }]);
    expect(order[0]).toBe(1);
  });

  it("paints the logo's top block last, over both lower blocks", () => {
    expect(paintOrder([ISO_PRESETS.cluster.blocks[2], ISO_PRESETS.cluster.blocks[0], ISO_PRESETS.cluster.blocks[1]])[2]).toBe(0);
  });
});

describe("frame", () => {
  it("rounds the block width, and the frame's, to an even number of pixels", () => {
    // 24 for the block, 3 of room, 1 more to make it even.
    expect(frame([{ x: 0, y: 0 }], 23).width).toBe(28);
    expect(frame([{ x: 0, y: 0 }], 24).width).toBe(28);
    for (const name of Object.keys(ISO_PRESETS) as (keyof typeof ISO_PRESETS)[]) {
      for (const size of [16, 22, 25, 40]) expect(frame(ISO_PRESETS[name].blocks, size).width % 2).toBe(0);
    }
  });

  it("keeps every preset on the grid's columns", () => {
    for (const preset of Object.values(ISO_PRESETS)) {
      for (const b of preset.blocks as readonly { x: number; y: number }[]) expect(Math.abs(b.x - b.y - Math.round(b.x - b.y))).toBeLessThan(1e-9);
    }
  });

  it("keeps the drawing's shape: the view box is the pixel size over the scale", () => {
    const f = frame(ISO_PRESETS.staircase.blocks, 30);
    const [, , w, h] = f.viewBox.split(" ").map(Number);
    expect(w / h).toBeCloseTo(f.width / f.height, 4);
    expect(w).toBeCloseTo(f.width / (30 / 26), 4);
  });

  it("puts vertical edges on half pixels", () => {
    const size = 20;
    const scale = size / 26;
    const f = frame(ISO_PRESETS.cluster.blocks, size);
    const minX = Number(f.viewBox.split(" ")[0]);
    for (const x of [-26, -13, 0, 13, 26]) {
      const px = (x - minX) * scale;
      expect(px % 1).toBeCloseTo(0.5, 3);
    }
  });
});

describe("idleBlock", () => {
  it("moves the block the layout marks accent", () => {
    expect(idleBlock(ISO_PRESETS.toppled.blocks)).toBe(2);
    // The staircase's accent is its top block, so an outline staircase lifts its top.
    expect(ISO_PRESETS.staircase.blocks[idleBlock(ISO_PRESETS.staircase.blocks)]).toMatchObject({ z: 2 });
  });

  it("else moves the last block painted, and none of no blocks", () => {
    expect(idleBlock([{ x: 0, y: 0, z: 1 }, { x: 0, y: 0 }])).toBe(0);
    expect(idleBlock([])).toBe(-1);
  });
});

describe("pixelsToUnits", () => {
  it("keeps a motion the same on screen at every size", () => {
    expect(pixelsToUnits(2.5, 26)).toBe(2.5);
    // The frame's pixels per drawing unit, times the units, is the pixels.
    for (const size of [18, 24, 28, 40]) {
      const f = frame([{ x: 0, y: 0 }], size);
      const pxPerUnit = f.width / Number(f.viewBox.split(" ")[2]);
      expect(pixelsToUnits(2.5, size) * pxPerUnit).toBeCloseTo(2.5, 2);
    }
  });
});

describe("pyramid", () => {
  it("grows like the logo, then as a pyramid", () => {
    expect(pyramid(0)).toEqual([]);
    expect(pyramid(1)).toEqual([{ x: 0, y: 0 }]);
    expect(pyramid(2)).toHaveLength(2);
    expect(pyramid(3).map(({ x, y, z }) => [x, y, z])).toEqual([
      [0, 1, 0],
      [1, 0, 0],
      [0.5, 0.5, 1],
    ]);
    expect(pyramid(6).at(-1)).toEqual({ x: 1, y: 1, z: 2 });
  });

  it("draws at most PYRAMID_MAX blocks", () => {
    expect(pyramid(40)).toHaveLength(PYRAMID_MAX);
    expect(pyramid(-1)).toEqual([]);
  });

  it("returns copies, so a caller can colour them", () => {
    const a = pyramid(3);
    a[0].kind = "vm";
    expect(pyramid(3)[0].kind).toBeUndefined();
  });
});
