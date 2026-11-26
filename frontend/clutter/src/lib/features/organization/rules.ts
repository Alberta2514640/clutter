// The limits on an organization's (a team's) name and description, owned once
// for every form that creates or renames one. The database holds a name of up
// to 32 characters and a description of up to 300
// (backend/storage-sql/application-tables.sql); the 4-character minimum is the
// website's rule, which A's handlers check too (generic.OrganizationNameOK).

export const ORGANIZATION_NAME_MIN = 4;
export const ORGANIZATION_NAME_MAX = 32;
export const ORGANIZATION_DESCRIPTION_MAX = 300;

export const organizationNameHint = `Use ${ORGANIZATION_NAME_MIN} to ${ORGANIZATION_NAME_MAX} characters.`;
export const organizationNamePlaceholder = "For example, Northwind Labs";
export const organizationDescriptionHint = `Optional. Up to ${ORGANIZATION_DESCRIPTION_MAX} characters, with no double spaces.`;
// A's create and rename refuse a name the person already uses for another team (409).
export const organizationNameTaken = "You already have a team with that name. Choose another name.";

// organizationNameProblem returns what is wrong with a name, or null.
// teamDeleteGoes is what the team delete confirmation says goes with the
// team, after its name: its projects (never "0 projects"), diagrams and
// connections.
export function teamDeleteGoes(projects: number): string {
  if (projects === 0) return ", its diagrams and its connections";
  return `, its ${projects === 1 ? "project" : `${projects} projects`}, its diagrams and its connections`;
}

export function organizationNameProblem(name: string): string | null {
  const trimmed = name.trim();
  if (trimmed.length < ORGANIZATION_NAME_MIN || trimmed.length > ORGANIZATION_NAME_MAX) return organizationNameHint;
  return null;
}

// defaultTeamName is the name onboarding suggests: "<first name>'s team" from
// the Google profile, or "My team" when there is no usable first name or the
// result would not fit.
export function defaultTeamName(displayName: string | null | undefined): string {
  const first = (displayName ?? "").trim().split(/\s+/)[0] ?? "";
  // The sign-in mapping writes "Unknown User" when Google gives no name.
  if (!first || displayName?.trim() === "Unknown User") return "My team";
  const name = `${first}'s team`;
  return organizationNameProblem(name) ? "My team" : name;
}

// The name of an AWS account link. A's stack URL handler refuses any other
// name; it owns the rule
// (backend/api/cloudformation/stack-url-generator/internal/validation.go).
export const AWS_ACCOUNT_NAME_MIN = 4;
export const AWS_ACCOUNT_NAME_MAX = 32;
export const awsAccountNameHint = `Use ${AWS_ACCOUNT_NAME_MIN} to ${AWS_ACCOUNT_NAME_MAX} letters and digits, with no spaces.`;

// awsAccountNameProblem returns what is wrong with an AWS account link's name, or null.
export function awsAccountNameProblem(name: string): string | null {
  const trimmed = name.trim();
  if (trimmed.length < AWS_ACCOUNT_NAME_MIN || trimmed.length > AWS_ACCOUNT_NAME_MAX || !/^[A-Za-z0-9]+$/.test(trimmed)) {
    return awsAccountNameHint;
  }
  return null;
}

// The role Clutter's CloudFormation stack creates. A copy of
// generic.ParseStackRoleARN (backend/api/generic/role_arn.go), which owns the
// shape and refuses any other ARN.
export const awsRoleArnPattern = /^arn:aws:iam::\d{12}:role\/AllowClutterToDeployTerraformRole-[a-z0-9]{8}$/;
export const awsRoleArnExample = "arn:aws:iam::123456789012:role/AllowClutterToDeployTerraformRole-ab12cd34";

// organizationDescriptionProblem returns what is wrong with a description, or null.
export function organizationDescriptionProblem(description: string): string | null {
  const trimmed = description.trim();
  if (trimmed.length === 0) return null;
  if (trimmed.length > ORGANIZATION_DESCRIPTION_MAX) return `Use ${ORGANIZATION_DESCRIPTION_MAX} characters or fewer.`;
  if (trimmed.includes("  ")) return "Remove the double spaces.";
  return null;
}
