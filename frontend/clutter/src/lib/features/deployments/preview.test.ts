import { describe, expect, it } from "vitest";
import { previewGroups } from "./preview";

describe("previewGroups", () => {
  it("keeps a one-cloud deployment's files in one unnamed group, sorted", () => {
    expect(previewGroups(["main.tf", "iam.tf"])).toEqual([
      {
        connection: null,
        files: [
          { path: "iam.tf", name: "iam.tf" },
          { path: "main.tf", name: "main.tf" },
        ],
      },
    ]);
  });

  it("groups a split deployment's files by connection, cutting at the last slash", () => {
    const groups = previewGroups(["Prod AWS/main.tf", "Lab/Edge/main.tf", "Prod AWS/iam.tf"]);
    expect(groups.map((g) => g.connection)).toEqual(["Lab/Edge", "Prod AWS"]);
    expect(groups[1].files.map((f) => f.name)).toEqual(["iam.tf", "main.tf"]);
    expect(groups[0].files[0].path).toBe("Lab/Edge/main.tf");
  });

  it("gives no groups for no files", () => {
    expect(previewGroups([])).toEqual([]);
  });
});
