import { describe, expect, it } from "vitest";
import { blockNameExample, RESOURCE_NAME_RULE, resourceNameError } from "./resourceName";

describe("resourceNameError", () => {
  it("accepts the names the backend accepts", () => {
    for (const name of ["test-lambda", "orders_table", "a", "web-1", "a".repeat(64)]) {
      expect(resourceNameError(name)).toBeNull();
    }
  });

  it("refuses the names the backend refuses", () => {
    for (const name of ["1api", "-api", "_api", "Files", "web server", 'files"', "${var.x}", "a".repeat(65)]) {
      expect(resourceNameError(name)).toBe(RESOURCE_NAME_RULE);
    }
  });

  it("leaves an empty name to the required check", () => {
    expect(resourceNameError("")).toBeNull();
  });

  it("says a name starts with a lowercase letter", () => {
    expect(RESOURCE_NAME_RULE).toContain("Start with a lowercase letter");
  });

  it("gives an example that fits the block, never a Lambda for a firewall", () => {
    const msg = resourceNameError("Web FW", blockNameExample("firewall"));
    expect(msg).toContain("for example web-fw.");
    expect(msg).not.toContain("lambda");
    expect(blockNameExample("something-new")).toBe("my-block");
  });
});
