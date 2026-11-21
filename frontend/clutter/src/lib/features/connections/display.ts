// How the connection flow presents each cloud beyond what its manifest says:
// which clouds a build offers, each cloud card's one line, and field help
// that replaces the manifest's. A connector this file does not list gets the
// generic wording from its credential spec.

// Relative imports: Vitest runs this file without the "@/" alias.
import type { CatalogEntry } from "../catalog/types";
import type { ConnectionStatus } from "./types";

type Presentation = {
  // Offered only in a development build.
  devOnly?: boolean;
  // The cloud card's line under the cloud's name.
  summary?: string;
  // Help that replaces the manifest's, by credential field name.
  fieldHelp?: Record<string, string>;
};

const PRESENTATION: Record<string, Presentation> = {
  aws: { summary: "Your own account, through a role you create there" },
  "general-openstack": { summary: "VEXXHOST and other OpenStack clouds" },
  // B's mock connector builds nothing. Its manifest's token help names the
  // one token it accepts, which no screen shows.
  mock: {
    devOnly: true,
    summary: "Test cloud (builds nothing, local only)",
    fieldHelp: { token: "The test token is in frontend/clutter/README.md." },
  },
};

// offeredClouds is the catalog a build shows: a development-only cloud is
// left out of every other build.
export function offeredClouds(catalog: CatalogEntry[], developmentBuild: boolean): CatalogEntry[] {
  return catalog.filter((e) => developmentBuild || !PRESENTATION[e.manifest.connector]?.devOnly);
}

// cloudSummary is the one line under a cloud's name on its card.
export function cloudSummary(entry: CatalogEntry): string {
  const own = PRESENTATION[entry.manifest.connector]?.summary;
  if (own) return own;
  if (entry.credentials.linked_only) return "Link your account, or use one you linked";
  if (entry.has_linked_accounts) return "Use an account you linked, or enter its details";
  if (entry.credentials.kind === "borrowed") return "Borrowed access";
  return "Credentials stored encrypted";
}

// credentialHelp is the help shown under a credential field.
export function credentialHelp(connector: string, field: string, manifestHelp?: string): string | undefined {
  return PRESENTATION[connector]?.fieldHelp?.[field] ?? manifestHelp;
}

// preferredCloud is the cloud the flow opens on: the one asked for, else the
// viewer's first cloud from onboarding, when the build offers it.
export function preferredCloud(offered: CatalogEntry[], ...wanted: (string | null | undefined)[]): string | null {
  for (const w of wanted) {
    if (w && offered.some((e) => e.manifest.connector === w)) return w;
  }
  return null;
}

// connectionStatusLabel is the one word every screen uses for a connection's
// last check: "Verified" when it worked, "Not working" when it did not.
export function connectionStatusLabel(status: ConnectionStatus): string {
  return status === "working" ? "Verified" : "Not working";
}
