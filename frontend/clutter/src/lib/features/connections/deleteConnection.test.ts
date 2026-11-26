import { describe, expect, it } from "vitest";
import { AdapterApiError } from "../../adapterApi";
import { deleteErrorMessage, deleteWarning, holdingDiagrams } from "./deleteConnection";

describe("holdingDiagrams", () => {
  const projects = [{ id: "p1", diagrams: [{ id: "d1", name: "shop" }, { id: "d2", name: "lab" }] }];
  const row = (over: Partial<Parameters<typeof holdingDiagrams>[1][number]>) => ({
    connection_id: "c1",
    diagram_id: "d1",
    project_id: "p1",
    status: "live" as const,
    placements: null,
    ...over,
  });

  it("names each diagram whose deployment on the connection may hold resources, once, with its Deploy link", () => {
    const rows = [row({}), row({ status: "failed" }), row({ diagram_id: "d2", connection_id: "c2", placements: [{ connection_id: "c1" }] })];
    expect(holdingDiagrams("c1", rows, projects)).toEqual([
      { diagramId: "d1", name: "shop", href: "/projects/p1/diagram/d1?panel=deploy" },
      { diagramId: "d2", name: "lab", href: "/projects/p1/diagram/d2?panel=deploy" },
    ]);
  });

  it("leaves out new and destroyed deployments, other connections, and names an unknown diagram plainly", () => {
    expect(holdingDiagrams("c1", [row({ status: "new" }), row({ status: "destroyed" }), row({ connection_id: "c9" })], projects)).toEqual([]);
    expect(holdingDiagrams("c1", [row({ diagram_id: "d9" })], projects)[0].name).toBe("a diagram");
  });
});

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
