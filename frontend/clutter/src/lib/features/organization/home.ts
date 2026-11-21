// Where a signed-in person starts, owned once for sign-in, Projects and
// onboarding: a person with a team starts on Projects; a person without one
// starts on Welcome (onboarding step 1).

export const PROJECTS_PATH = "/dashboard";
export const WELCOME_PATH = "/onboarding";

// homePath is the page a signed-in person with these teams starts on.
export function homePath(organizations: readonly unknown[] | null | undefined): string {
  return (organizations?.length ?? 0) > 0 ? PROJECTS_PATH : WELCOME_PATH;
}
