import { describe, expect, it } from "vitest";
import { AdapterApiError } from "../../adapterApi";
import type { OrganizationAwsAccount } from "../organization/types";
import { accountsInOrder, connectFailure, isFakeLocalRole, LOCAL_AWS_REFUSED, resumeTarget, roleArnProblem } from "./awsLink";

const account = (id: string, name: string, status: string, role_arn: string | null = null): OrganizationAwsAccount => ({
  id,
  account_name: name,
  status,
  role_arn,
});

describe("roleArnProblem", () => {
  it("has nothing to say about an empty field or the stack's role", () => {
    expect(roleArnProblem("  ")).toBeNull();
    expect(roleArnProblem(" arn:aws:iam::123456789012:role/AllowClutterToDeployTerraformRole-ab12cd34 ")).toBeNull();
  });

  it("names what is wrong with anything else", () => {
    expect(roleArnProblem("hello")).toMatch(/starts with arn:aws:iam::/);
    expect(roleArnProblem("arn:aws:iam::123456789012:role/SomeOtherRole")).toMatch(/not the role Clutter's stack creates/);
  });
});

describe("accountsInOrder and resumeTarget", () => {
  const accounts = [
    account("3", "Zeta", "incomplete"),
    account("1", "Beta", "complete", "arn"),
    account("2", "Alpha", "incomplete"),
    account("4", "Gamma", "revoked"),
  ];

  it("lists linked accounts first, then unfinished, each by name", () => {
    expect(accountsInOrder(accounts).map((a) => a.account_name)).toEqual(["Beta", "Alpha", "Zeta", "Gamma"]);
  });

  it("resumes the chosen unfinished link, else the first one", () => {
    expect(resumeTarget(accounts, "3")?.account_name).toBe("Zeta");
    expect(resumeTarget(accounts, "1")?.account_name).toBe("Alpha");
    expect(resumeTarget(accounts, null)?.account_name).toBe("Alpha");
    expect(resumeTarget([accounts[1]], null)).toBeNull();
  });
});

describe("local fake accounts", () => {
  const refused = new AdapterApiError(422, "verification_failed", "AWS did not let Clutter use this role. If Clutter's CloudFormation stack was deleted ...");

  it("knows the devserver's fake role", () => {
    expect(isFakeLocalRole("arn:aws:iam::123456789012:role/AllowClutterToDeployTerraformRole-abc")).toBe(true);
    expect(isFakeLocalRole("arn:aws:iam::210987654321:role/AllowClutterToDeployTerraformRole-abc")).toBe(false);
    expect(isFakeLocalRole(null)).toBe(false);
  });

  it("explains a refused fake account without blaming a deleted stack", () => {
    expect(connectFailure(refused, true)).toBe(LOCAL_AWS_REFUSED);
    expect(connectFailure(refused, false)).toBe(refused.message);
    expect(connectFailure(new AdapterApiError(409, "name_taken", "That name is taken."), true)).toBe("That name is taken.");
  });
});
