// Relative import: Vitest runs this file without the "@/" alias.
import type { Organization } from "./types";

// sortOrganizations orders teams by name (ignoring case), then by id, so the
// switcher and the "first team" fallback never change between loads.
export function sortOrganizations<T extends Pick<Organization, "id" | "name">>(organizations: readonly T[]): T[] {
  return [...organizations].sort(
    (a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: "base" }) || a.id.localeCompare(b.id),
  );
}
