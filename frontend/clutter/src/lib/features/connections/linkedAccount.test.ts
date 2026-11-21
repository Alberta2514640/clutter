import { describe, expect, it } from "vitest";
import { startRegion } from "./linkedAccount";

describe("startRegion", () => {
  const regions = ["ap-northeast-1", "ca-central-1", "us-east-1"];

  it("starts on the region the link flow opens in, when the cloud offers it", () => {
    expect(startRegion(regions, "ca-central-1")).toBe("ca-central-1");
  });

  it("falls back to the cloud's first region, or none", () => {
    expect(startRegion(regions, "eu-west-1")).toBe("ap-northeast-1");
    expect(startRegion(regions)).toBe("ap-northeast-1");
    expect(startRegion([], "ca-central-1")).toBe("");
  });
});
