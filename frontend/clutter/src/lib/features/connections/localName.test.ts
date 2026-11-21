import { describe, expect, it } from "vitest";
import { localConnectionName } from "./localName";

describe("localConnectionName", () => {
  it("is Local and the cloud's short name, with a number only when the name is taken", () => {
    expect(localConnectionName("AWS", ["lab"])).toBe("Local AWS");
    expect(localConnectionName("OpenStack", ["Local OpenStack"])).toBe("Local OpenStack 2");
    expect(localConnectionName("AWS", ["Local AWS", "Local AWS 2"])).toBe("Local AWS 3");
  });

  it("stays within B's 64 characters", () => {
    const long = "x".repeat(70);
    expect(localConnectionName(long, []).length).toBe(64);
    expect(localConnectionName(long, [localConnectionName(long, [])]).endsWith(" 2")).toBe(true);
    expect(localConnectionName(long, [localConnectionName(long, [])]).length).toBe(64);
  });
});
