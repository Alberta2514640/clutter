// The manifest-to-display mapping: how B's screens name and mark a cloud and
// show a block on it. Each manifest owns its short name, logo key, and per
// block service name and icon key; this file only turns keys into the files
// under public/clouds/<connector>/ (sources and licences in that folder's
// README). A key that does not fit the pattern B checks resolves to nothing,
// so the screen falls back to the neutral icon.

// Relative imports: Vitest runs this file without the "@/" alias.
import type { BlockKind, CatalogEntry, Manifest } from "./types";

// connectors.ConnectorIDPattern and connectors.IconKeyPattern.
const CONNECTOR_ID = /^[a-z][a-z0-9-]{1,40}$/;
const ICON_KEY = /^[a-z0-9][a-z0-9-]{0,47}$/;

// iconSrc is the public path of a cloud's logo or icon, or null when there is none.
export function iconSrc(connector: string, key: string | null | undefined): string | null {
  if (!key || !CONNECTOR_ID.test(connector) || !ICON_KEY.test(key)) return null;
  return `/clouds/${connector}/${key}.svg`;
}

// How a cloud is named and marked.
export type CloudLook = {
  connector: string;
  title: string;
  shortName: string;
  logoSrc: string | null;
  // What the cloud is when its title does not say, or "".
  note?: string;
};

export function cloudLook(m: Manifest): CloudLook {
  return {
    connector: m.connector,
    title: m.title,
    shortName: m.display?.short_name || m.title,
    logoSrc: iconSrc(m.connector, m.display?.logo),
    ...(m.display?.note ? { note: m.display.note } : {}),
  };
}

// cloudTitleWithNote is the title the Connections page shows: "Mock cloud
// (for testing; builds nothing)".
export const cloudTitleWithNote = (c: CloudLook) => (c.note ? `${c.title} (${c.note})` : c.title);

// cloudLookFor names a connector from the catalog, or by its id when the
// catalog no longer offers it.
export function cloudLookFor(catalog: CatalogEntry[], connector: string): CloudLook {
  const entry = catalog.find((e) => e.manifest.connector === connector);
  return entry ? cloudLook(entry.manifest) : { connector, title: connector, shortName: connector, logoSrc: null };
}

// How one cloud shows a kind of block.
export type BlockLook = {
  // The cloud's service name, or the block's own title when the cloud does
  // not build it.
  name: string;
  iconSrc: string | null;
  offered: boolean;
};

export function blockLook(m: Manifest, kind: BlockKind, blockTitle: string): BlockLook {
  const b = (m.blocks ?? []).find((x) => x.kind === kind);
  if (!b || b.treatment === "removed") return { name: blockTitle, iconSrc: null, offered: false };
  return { name: b.display?.name || b.title, iconSrc: iconSrc(m.connector, b.display?.icon), offered: true };
}

// monogram is what stands in for a logo that will not load: the short name
// when it is three characters or fewer, else its first letter.
export function monogram(shortName: string): string {
  const s = shortName.trim();
  return s.length <= 3 ? s : s.slice(0, 1).toUpperCase();
}
