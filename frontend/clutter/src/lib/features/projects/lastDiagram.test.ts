import { describe, expect, it } from "vitest";
import { findLastDiagram } from "./lastDiagram";
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

describe("findLastDiagram", () => {
  const projects = [project("p1", [{ id: "d1", name: "web" }]), project("p2", [])];

  it("finds the last diagram while it is still in its project", () => {
    expect(findLastDiagram({ projectId: "p1", diagramId: "d1" }, projects)).toEqual({ project: projects[0], diagram: { id: "d1", name: "web" } });
  });

  it("finds nothing for a deleted diagram, a deleted project, or no memory", () => {
    expect(findLastDiagram({ projectId: "p2", diagramId: "d1" }, projects)).toBeNull();
    expect(findLastDiagram({ projectId: "p9", diagramId: "d1" }, projects)).toBeNull();
    expect(findLastDiagram(null, projects)).toBeNull();
    expect(findLastDiagram({ projectId: "p1", diagramId: "d1" }, undefined)).toBeNull();
  });
});
