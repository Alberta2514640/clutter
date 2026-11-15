// The canvas's rule for a resource name. The backend owns it:
// hclsafe.CheckName in backend/api/generic/hclsafe/hclsafe.go refuses any
// other name when it generates Terraform. Change both together.
export const RESOURCE_NAME_PATTERN = /^[a-z][a-z0-9_-]*$/;

// The backend's cap (hclsafe.MaxNameLength).
export const RESOURCE_NAME_MAX_LENGTH = 64;

// The name field stops typing here, which is stricter than the backend's cap.
export const RESOURCE_NAME_INPUT_MAX_LENGTH = 32;

export const RESOURCE_NAME_RULE =
  "Start with a lowercase letter, then use only lowercase letters, numbers, hyphens and underscores, for example test-lambda.";

// resourceNameError returns the rule when name breaks it, or null. An empty
// name is the required check's job, not this one's.
export function resourceNameError(name: string): string | null {
  if (name === "") return null;
  if (name.length > RESOURCE_NAME_MAX_LENGTH || !RESOURCE_NAME_PATTERN.test(name)) {
    return RESOURCE_NAME_RULE;
  }
  return null;
}
