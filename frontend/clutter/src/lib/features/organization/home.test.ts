import { describe, expect, it } from "vitest";
import { homePath, PROJECTS_PATH, WELCOME_PATH } from "./home";

describe("homePath", () => {
  it("sends a person with a team to Projects", () => {
    expect(homePath([{ id: "a" }])).toBe(PROJECTS_PATH);
    expect(homePath([{ id: "a" }, { id: "b" }])).toBe(PROJECTS_PATH);
  });

  it("sends a person without a team to Welcome", () => {
    expect(homePath([])).toBe(WELCOME_PATH);
    expect(homePath(null)).toBe(WELCOME_PATH);
    expect(homePath(undefined)).toBe(WELCOME_PATH);
  });
});
