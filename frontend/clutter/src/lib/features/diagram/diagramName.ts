// A diagram's name is capped by A's diagrams table.
export const DIAGRAM_NAME_MAX = 32;

// BLANK_DIAGRAM_NAME is what a new blank diagram is called until it is named.
export const BLANK_DIAGRAM_NAME = "Untitled diagram";

// diagramNameError returns why a typed diagram name cannot be used, or null.
export function diagramNameError(name: string): string | null {
  if (name.trim() === "") return "Give the diagram a name.";
  if (name.trim().length > DIAGRAM_NAME_MAX) return `Use at most ${DIAGRAM_NAME_MAX} characters.`;
  return null;
}

// uniqueDiagramName returns base when no diagram in the project has that
// name, else base with the lowest free number from 2 ("Web server 2"),
// cut so the whole fits DIAGRAM_NAME_MAX.
export function uniqueDiagramName(base: string, taken: string[]): string {
  const used = new Set(taken);
  const fit = (s: string, suffix = "") => s.slice(0, DIAGRAM_NAME_MAX - suffix.length).trimEnd() + suffix;
  const first = fit(base);
  if (!used.has(first)) return first;
  for (let i = 2; ; i++) {
    const name = fit(base, ` ${i}`);
    if (!used.has(name)) return name;
  }
}
