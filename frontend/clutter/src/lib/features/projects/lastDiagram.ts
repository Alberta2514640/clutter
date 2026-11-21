// The diagram last opened in a team (preferences store's lastDiagram), found
// in the team's projects as they are now. The Projects page's "Continue where
// you left off" and the "Getting started" checklist both link to it only
// while it still exists: a deleted diagram or project leaves no dead link.

// Relative imports: Vitest runs this file without the "@/" alias.
import type { DiagramSummary, Project } from "./types";

type LastDiagramRef = { projectId: string; diagramId: string };

export function findLastDiagram(
  last: LastDiagramRef | null | undefined,
  projects: readonly Project[] | null | undefined,
): { project: Project; diagram: DiagramSummary } | null {
  if (!last) return null;
  const project = (projects ?? []).find((p) => p.id === last.projectId);
  const diagram = project?.diagrams?.find((d) => d.id === last.diagramId);
  return project && diagram ? { project, diagram } : null;
}
