// "Get to your first deploy" (the editor) and "Getting started" (Projects):
// one checklist with four steps, owned here so both places agree on what is
// done, what comes next, when it is finished and hidden, and where each
// unfinished step leads.

// Relative imports: Vitest runs this file without the "@/" alias.
import { findLastDiagram } from "./lastDiagram";
import type { DiagramSummary, Project } from "./types";

export const CHECKLIST_STEPS = ["draw", "connect", "preview", "deploy"] as const;

export type StepId = (typeof CHECKLIST_STEPS)[number];

export type ChecklistProgress = { count: number; next: StepId | null; finished: boolean };

// checklistProgress counts the steps done and names the first one not done.
// The checklist is finished, and both places hide it, once a deployment is
// live: that is the first deploy it leads to.
export function checklistProgress(done: Record<StepId, boolean>): ChecklistProgress {
  return {
    count: CHECKLIST_STEPS.filter((s) => done[s]).length,
    next: CHECKLIST_STEPS.find((s) => !done[s]) ?? null,
    finished: done.deploy,
  };
}

export type StepLink = { label: string; href: string };

// The query that opens a project's New diagram dialog on arrival.
export const NEW_DIAGRAM_QUERY = "new=diagram";

// checklistLinks is where Projects' card sends each step: the diagram last
// worked on (or, with none remembered, the first diagram of the team's
// projects) at the Deploy view or the code panel. With no diagram at all,
// every diagram step opens the New diagram dialog of the remembered or first
// project, and with no project, the New project page.
export function checklistLinks(
  last: { projectId: string; diagramId: string } | null | undefined,
  projects: readonly Project[] | null | undefined,
): Record<"draw" | "preview" | "deploy", StepLink> {
  const list = projects ?? [];
  const target: { project: Project; diagram: DiagramSummary } | null =
    findLastDiagram(last, list) ??
    list.flatMap((p) => (p.diagrams ?? []).slice(0, 1).map((diagram) => ({ project: p, diagram })))[0] ??
    null;
  const home = list.find((p) => p.id === last?.projectId) ?? list[0] ?? null;
  const create: StepLink = home
    ? { label: "New diagram", href: `/projects/${home.id}/diagrams?${NEW_DIAGRAM_QUERY}` }
    : { label: "New project", href: "/projects/new" };
  if (!target) return { draw: create, preview: create, deploy: create };
  const at = (panel?: "code" | "deploy") => `/projects/${target.project.id}/diagram/${target.diagram.id}${panel ? `?panel=${panel}` : ""}`;
  return {
    draw: { label: `Open ${target.diagram.name}`, href: at() },
    preview: { label: "Preview the code", href: at("code") },
    deploy: { label: "Open Deploy", href: at("deploy") },
  };
}
