import { describe, expect, it } from "vitest";
import { FIRST_PROJECT_NAME, PROJECT_NAME_MAX, projectDeleteGoes, projectDeleteLine, projectDescriptionProblem, projectNameProblem } from "./rules";

describe("projectDeleteLine and projectDeleteGoes", () => {
  it("never says 0 diagrams, and words one diagram as one", () => {
    expect(projectDeleteLine(0)).toBe("This project has no diagrams. Deleting it changes nothing in your cloud.");
    expect(projectDeleteLine(1)).toBe("Deleting the project deletes its diagram for good. Nothing in your cloud changes.");
    expect(projectDeleteLine(3)).toBe("Deleting the project deletes its 3 diagrams for good. Nothing in your cloud changes.");
    expect(projectDeleteGoes(0)).toBe("");
    expect(projectDeleteGoes(1)).toBe(" and its diagram");
    expect(projectDeleteGoes(2)).toBe(" and its 2 diagrams");
  });
});

describe("projectNameProblem", () => {
  it("accepts a name that fits and is free", () => {
    expect(projectNameProblem("Web shop")).toBeNull();
    expect(projectNameProblem("  Web shop  ", ["Other"])).toBeNull();
    expect(projectNameProblem(FIRST_PROJECT_NAME)).toBeNull();
    expect(projectNameProblem("a".repeat(PROJECT_NAME_MAX))).toBeNull();
  });

  it("refuses an empty, long or taken name", () => {
    expect(projectNameProblem("   ")).toBe("Enter a project name.");
    expect(projectNameProblem("a".repeat(PROJECT_NAME_MAX + 1))).toMatch(/32 characters/);
    expect(projectNameProblem(" Web shop ", ["Web shop"])).toMatch(/has that name/);
  });
});

describe("projectDescriptionProblem", () => {
  it("allows up to 300 characters", () => {
    expect(projectDescriptionProblem("")).toBeNull();
    expect(projectDescriptionProblem("x".repeat(300))).toBeNull();
    expect(projectDescriptionProblem("x".repeat(301))).toMatch(/300/);
  });
});
