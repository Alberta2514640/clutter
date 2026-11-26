import { describe, expect, it } from "vitest";
import { SHOTS, SIGNIN_SHOTS, VIEW_SHOTS } from "./shots";

// The media relabel a local test connection as AWS and OpenStack (shots.ts),
// so no alt text may say a real cloud built what the picture shows.
const alts = [...Object.values(SHOTS), ...Object.values(SIGNIN_SHOTS), ...VIEW_SHOTS.map((v) => v.shot)].map((s) => s.alt);

describe("landing media alt text", () => {
  it("calls the relabelled connection an example wherever it names it", () => {
    for (const alt of alts.filter((a) => a.includes("Northwind AWS"))) {
      expect(alt).toMatch(/example/i);
    }
  });

  it("never places a deployment on a real cloud or region", () => {
    for (const alt of alts) {
      expect(alt).not.toMatch(/on Amazon Web Services in|ca-central-1|in RegionOne/);
    }
  });
});
