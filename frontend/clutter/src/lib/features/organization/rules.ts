// The limits on an organization's name and description, owned once for both
// forms that create one. The database holds a name of up to 32 characters and
// a description of up to 300 (backend/storage-sql/application-tables.sql); the
// 4-character minimum is the website's own rule.

export const ORGANIZATION_NAME_MIN = 4;
export const ORGANIZATION_NAME_MAX = 32;
export const ORGANIZATION_DESCRIPTION_MAX = 300;

export const organizationNameHint = `Use ${ORGANIZATION_NAME_MIN} to ${ORGANIZATION_NAME_MAX} characters.`;
// Shown under the name on both create forms: the backend has no rename.
export const organizationNameCreateHint = `${organizationNameHint} You cannot change it later.`;
export const organizationNamePlaceholder = "e.g. Clutter Labs";
export const organizationDescriptionHint = `Optional. Up to ${ORGANIZATION_DESCRIPTION_MAX} characters, with no double spaces.`;

// organizationNameProblem returns what is wrong with a name, or null.
export function organizationNameProblem(name: string): string | null {
  const trimmed = name.trim();
  if (trimmed.length < ORGANIZATION_NAME_MIN || trimmed.length > ORGANIZATION_NAME_MAX) return organizationNameHint;
  return null;
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
