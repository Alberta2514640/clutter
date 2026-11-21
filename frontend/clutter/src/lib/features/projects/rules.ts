// The limits on a project's name and description, owned once for every form
// that creates or renames one. A's projects table holds a name of up to 32
// characters, unique in its organization, and a description of up to 300
// (backend/storage-sql/application-tables.sql). A's create answers a taken
// name with a bare 500, so the forms check the names they already have.

export const PROJECT_NAME_MAX = 32;
export const PROJECT_DESCRIPTION_MAX = 300;

// The project onboarding makes for a new team's first diagram.
export const FIRST_PROJECT_NAME = "My first project";

// projectNameProblem returns what is wrong with a name, or null. taken holds
// the names of the team's other projects.
export function projectNameProblem(name: string, taken: readonly string[] = []): string | null {
  const trimmed = name.trim();
  if (!trimmed) return "Enter a project name.";
  if (trimmed.length > PROJECT_NAME_MAX) return `Use ${PROJECT_NAME_MAX} characters or fewer.`;
  if (taken.includes(trimmed)) return "Another project in this team has that name. Choose another name.";
  return null;
}

// projectDeleteLine is the Danger zone's line for a project with this many
// diagrams, and projectDeleteGoes what the confirmation says goes with it
// ("and its diagram"), so no screen says "0 diagrams".
export function projectDeleteLine(diagrams: number): string {
  if (diagrams === 0) return "This project has no diagrams. Deleting it changes nothing in your cloud.";
  if (diagrams === 1) return "Deleting the project deletes its diagram for good. Nothing in your cloud changes.";
  return `Deleting the project deletes its ${diagrams} diagrams for good. Nothing in your cloud changes.`;
}

export function projectDeleteGoes(diagrams: number): string {
  if (diagrams === 0) return "";
  return diagrams === 1 ? " and its diagram" : ` and its ${diagrams} diagrams`;
}

// projectDescriptionProblem returns what is wrong with a description, or null.
export function projectDescriptionProblem(description: string): string | null {
  if (description.trim().length > PROJECT_DESCRIPTION_MAX) return `Use ${PROJECT_DESCRIPTION_MAX} characters or fewer.`;
  return null;
}
