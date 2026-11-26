import { describe, expect, it } from "vitest";
import { checklistLinks, checklistProgress } from "./checklist";
import type { Project } from "./types";

const project = (id: string, diagrams: { id: string; name: string }[]): Project => ({
  id,
  organizationId: "o",
  creatorId: "u",
  name: id,
  diagrams,
  description: "",
  createdAt: "",
  createdBy: "",
});

describe("checklistProgress", () => {
  it("counts the steps and names the first one not done", () => {
    expect(checklistProgress({ draw: true, connect: false, preview: true, deploy: false })).toEqual({ count: 2, next: "connect", finished: false });
  });

  it("is finished, for the editor and Projects alike, once a deployment is live", () => {
    expect(checklistProgress({ draw: true, connect: true, preview: false, deploy: true })).toEqual({ count: 3, next: "preview", finished: true });
    expect(checklistProgress({ draw: true, connect: true, preview: true, deploy: false }).finished).toBe(false);
  });
});

describe("checklistLinks", () => {
  const projects = [project("p1", []), project("p2", [{ id: "d2", name: "shop" }]), project("p3", [{ id: "d3", name: "lab" }])];

  it("opens the last diagram at each view", () => {
    expect(checklistLinks({ projectId: "p3", diagramId: "d3" }, projects)).toEqual({
      draw: { label: "Open lab", href: "/projects/p3/diagram/d3" },
      preview: { label: "Preview the code", href: "/projects/p3/diagram/d3?panel=code" },
      deploy: { label: "Open Deploy", href: "/projects/p3/diagram/d3?panel=deploy" },
    });
  });

  it("falls back to the team's first diagram when the last one is gone or unknown", () => {
    expect(checklistLinks({ projectId: "p3", diagramId: "gone" }, projects).deploy.href).toBe("/projects/p2/diagram/d2?panel=deploy");
    expect(checklistLinks(null, projects).draw.label).toBe("Open shop");
  });

  it("opens the New diagram dialog with no diagram, and New project with no project", () => {
    const empty = checklistLinks({ projectId: "p1", diagramId: "x" }, [project("p0", []), project("p1", [])]);
    expect(empty.draw).toEqual({ label: "New diagram", href: "/projects/p1/diagrams?new=diagram" });
    expect(empty.preview).toEqual(empty.draw);
    expect(checklistLinks(null, []).deploy).toEqual({ label: "New project", href: "/projects/new" });
  });
});
