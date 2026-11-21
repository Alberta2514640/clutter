import { describe, expect, it } from "vitest";
import { sortOrganizations } from "./sort";

describe("sortOrganizations", () => {
  it("orders by name ignoring case, then by id", () => {
    const teams = [
      { id: "3", name: "beta" },
      { id: "2", name: "Alpha" },
      { id: "1", name: "alpha" },
      { id: "4", name: "Gamma" },
    ];
    expect(sortOrganizations(teams).map((t) => t.id)).toEqual(["1", "2", "3", "4"]);
  });

  it("gives the same order whatever order the API answers in", () => {
    const a = [
      { id: "b", name: "Lab" },
      { id: "a", name: "Lab" },
      { id: "c", name: "Corner" },
    ];
    expect(sortOrganizations(a)).toEqual(sortOrganizations([...a].reverse()));
  });

  it("leaves the input unchanged", () => {
    const teams = [
      { id: "2", name: "B" },
      { id: "1", name: "A" },
    ];
    sortOrganizations(teams);
    expect(teams[0].id).toBe("2");
  });
});
