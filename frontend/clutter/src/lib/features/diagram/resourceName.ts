// The canvas's rule for a resource name. The backend owns it:
// hclsafe.CheckName in backend/api/generic/hclsafe/hclsafe.go refuses any
// other name when it generates Terraform. Change both together.
export const RESOURCE_NAME_PATTERN = /^[a-z][a-z0-9_-]*$/;

// The backend's cap (hclsafe.MaxNameLength).
export const RESOURCE_NAME_MAX_LENGTH = 64;

// The name field stops typing here, which is stricter than the backend's cap.
export const RESOURCE_NAME_INPUT_MAX_LENGTH = 32;

// resourceNameRule says the rule with an example that fits the thing being
// named: A's editor names Lambdas, so its example is one; B's editor gives
// each block kind its own ("web-1" for a VM).
export const resourceNameRule = (example: string) =>
  `Start with a lowercase letter, then use only lowercase letters, numbers, hyphens and underscores, for example ${example}.`;

export const RESOURCE_NAME_RULE = resourceNameRule("test-lambda");

// resourceNameError returns the rule when name breaks it, or null. An empty
// name is the required check's job, not this one's.
export function resourceNameError(name: string, example = "test-lambda"): string | null {
  if (name === "") return null;
  if (name.length > RESOURCE_NAME_MAX_LENGTH || !RESOURCE_NAME_PATTERN.test(name)) {
    return resourceNameRule(example);
  }
  return null;
}

// The example B's editor gives for a block kind's name.
const BLOCK_NAME_EXAMPLES: Record<string, string> = {
  vm: "web-1",
  network: "private-net",
  firewall: "web-fw",
  volume: "data-disk",
  bucket: "uploads",
  function: "orders-handler",
  table: "orders",
  http_api: "public-api",
};

export const blockNameExample = (kind: string) => BLOCK_NAME_EXAMPLES[kind] ?? "my-block";
