import { describe, expect, it } from "vitest";
import { joinAnd, plural } from "./text";

describe("plural", () => {
  it("counts a noun", () => {
    expect(plural(0, "block", "blocks")).toBe("0 blocks");
    expect(plural(1, "block", "blocks")).toBe("1 block");
    expect(plural(3, "block runs", "blocks run")).toBe("3 blocks run");
  });
});

describe("joinAnd", () => {
  it("lists names in a sentence", () => {
    expect(joinAnd([])).toBe("");
    expect(joinAnd(["a"])).toBe("a");
    expect(joinAnd(["a", "b"])).toBe("a and b");
    expect(joinAnd(["a", "b", "c"])).toBe("a, b and c");
  });
});
