import { describe, expect, it } from "vitest";
import { awsAccountNameHint, awsAccountNameProblem, awsRoleArnExample, awsRoleArnPattern, defaultTeamName, organizationNameProblem, teamDeleteGoes } from "./rules";

describe("teamDeleteGoes", () => {
  it("never says 0 projects, and words one project as one", () => {
    expect(teamDeleteGoes(0)).toBe(", its diagrams and its connections");
    expect(teamDeleteGoes(1)).toBe(", its project, its diagrams and its connections");
    expect(teamDeleteGoes(3)).toBe(", its 3 projects, its diagrams and its connections");
  });
});

describe("defaultTeamName", () => {
  it("uses the first name from the profile", () => {
    expect(defaultTeamName("Ada Lovelace")).toBe("Ada's team");
    expect(defaultTeamName("  Grace  ")).toBe("Grace's team");
  });

  it("falls back to My team without a usable name", () => {
    for (const name of [undefined, null, "", "   ", "Unknown User"]) expect(defaultTeamName(name)).toBe("My team");
  });

  it("always suggests a name the rules accept", () => {
    for (const name of ["Al Smith", "A", "Bartholomew-Christopherson-Smythe Jones"]) {
      expect(organizationNameProblem(defaultTeamName(name))).toBeNull();
    }
    expect(defaultTeamName("Bartholomew-Christopherson-Smythe Jones")).toBe("My team");
  });
});

describe("awsAccountNameProblem", () => {
  it("accepts 4 to 32 letters and digits", () => {
    for (const name of ["Prod", "Production2", " Staging ", "a".repeat(32)]) expect(awsAccountNameProblem(name)).toBeNull();
  });

  it("refuses what A's stack URL handler refuses, with the rule", () => {
    for (const name of ["abc", "a".repeat(33), "My account", "prod-1", "prod_1", "Prodé", ""]) {
      expect(awsAccountNameProblem(name)).toBe(awsAccountNameHint);
    }
  });
});

describe("awsRoleArnPattern", () => {
  it("accepts only the role Clutter's stack creates", () => {
    expect(awsRoleArnPattern.test(awsRoleArnExample)).toBe(true);
    for (const arn of [
      "arn:aws:iam::123456789012:role/ClutterDeployRole",
      "arn:aws:iam::123456789012:role/Other-ab12cd34",
      "arn:aws:iam::123456789012:role/path/AllowClutterToDeployTerraformRole-ab12cd34",
      "arn:aws:iam::123456789012:role/AllowClutterToDeployTerraformRole-AB12CD34",
      "arn:aws:iam::123456789012:role/AllowClutterToDeployTerraformRole-abc",
    ]) {
      expect(awsRoleArnPattern.test(arn)).toBe(false);
    }
  });
});
