import { describe, expect, it } from "vitest";
import { AdapterApiError } from "../../adapterApi";
import { deleteErrorMessage, deleteWarning } from "./deleteConnection";

describe("deleteWarning", () => {
  it("asks plainly when no diagram deploys to the connection", () => {
    expect(deleteWarning(0)).toBe("No diagram deploys to this connection. Delete it?");
  });

  it("says a built deployment must be destroyed first, and one not built is removed", () => {
    expect(deleteWarning(1)).toContain("If its deployment is built, destroy it first");
    expect(deleteWarning(1)).toContain("If not, deleting removes the deployment");
    expect(deleteWarning(3)).toMatch(/^3 diagrams deploy to this connection\. Destroy the deployments that are built first/);
    expect(deleteWarning(3)).toContain("Deleting removes the rest");
  });
});

describe("deleteErrorMessage", () => {
  it("words B's refusal while a deployment is still built", () => {
    const err = new AdapterApiError(409, "live_deployments", "This connection has live deployments. Destroy them first.");
    expect(deleteErrorMessage(err)).toBe(
      "A deployment on this connection is still built. Destroy it on its diagram's Deploy tab first.",
    );
  });

  it("keeps B's own message for any other refusal", () => {
    expect(deleteErrorMessage(new AdapterApiError(404, "not_found", "Not found."))).toBe("Not found.");
  });
});
