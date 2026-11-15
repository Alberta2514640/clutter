import { describe, expect, it } from "vitest";
import { chosenLinkedAccount, startRegion } from "./linkedAccount";

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

describe("chosenLinkedAccount", () => {
  const prod = { id: "1", name: "Production" };
  const lab = { id: "2", name: "Lab" };

  it("chooses an account just linked inline once B lists it", () => {
    expect(chosenLinkedAccount([prod], null, "Lab")).toBeNull();
    expect(chosenLinkedAccount([prod, lab], null, "Lab")).toBe(lab);
  });

  it("keeps the user's own pick over the account just linked", () => {
    expect(chosenLinkedAccount([prod, lab], prod, "Lab")).toBe(prod);
    expect(chosenLinkedAccount([prod, lab], null, null)).toBeNull();
  });
});
