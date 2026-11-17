import { describe, expect, it } from "vitest";
import { awsAccountNameHint, awsAccountNameProblem, awsRoleArnExample, awsRoleArnPattern } from "./rules";

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
