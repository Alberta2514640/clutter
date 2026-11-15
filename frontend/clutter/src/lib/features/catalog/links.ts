// The link rules of B's editor: which lines a person may draw between two
// blocks, why a line is refused, and what a line means and is good for. The
// manifests' edge lists own every rule, meaning, why and refusal line; this
// file only reads them for the clouds in play: the clouds this diagram deploys
// to or the organization connects to, or every cloud in the catalog when
// there are none yet.

// Relative imports: Vitest runs this file without the "@/" alias.
import { NETWORK_KIND, VM_KIND } from "../diagram/blocksDocument";
import type { BlockKind, CatalogEntry, EdgeSpec, Manifest } from "./types";

// manifestsInPlay are the manifests of the given clouds, or of the whole
// catalog when none of them is in it.
export function manifestsInPlay(catalog: CatalogEntry[], connectors: string[]): Manifest[] {
  const chosen = catalog.filter((e) => connectors.includes(e.manifest.connector)).map((e) => e.manifest);
  return chosen.length > 0 ? chosen : catalog.map((e) => e.manifest);
}

const edgesOf = (manifests: Manifest[], source: BlockKind, target: BlockKind): EdgeSpec[] =>
  manifests.flatMap((m) => (m.edges ?? []).filter((e) => e.source === source && e.target === target));

// One end of a line, and what the canvas knows of it.
export type LinkEnd = { id: string; kind: BlockKind };

export type LinkCheck = { ok: true } | { ok: false; reason: string };

const article = (title: string) => (/^[aeiou]/i.test(title) ? "An" : "A");

// linkCheck says whether a line from one block to another may be drawn, and
// when not, why, in one short line. kindOf finds the kind of a block the
// existing lines name.
export function linkCheck(
  manifests: Manifest[],
  from: LinkEnd,
  to: LinkEnd,
  lines: { source: string; target: string }[],
  kindOf: (id: string) => BlockKind | undefined,
  titleOf: (kind: BlockKind) => string,
): LinkCheck {
  if (from.id === to.id) return { ok: false, reason: "A block cannot link to itself." };
  if (from.kind === VM_KIND && to.kind === NETWORK_KIND) return { ok: false, reason: "Drag the VM into the network box instead." };
  if (lines.some((l) => l.source === from.id && l.target === to.id)) return { ok: false, reason: "These two are already linked." };
  const specs = edgesOf(manifests, from.kind, to.kind);
  if (specs.length === 0) {
    if (edgesOf(manifests, to.kind, from.kind).length > 0) {
      return { ok: false, reason: `Draw it the other way, from the ${titleOf(to.kind)} to the ${titleOf(from.kind)}.` };
    }
    return { ok: false, reason: `${article(titleOf(from.kind))} ${titleOf(from.kind)} cannot link to ${article(titleOf(to.kind)).toLowerCase()} ${titleOf(to.kind)}.` };
  }
  // A rule holds when every cloud in play that draws this line keeps it.
  if (specs.every((s) => s.single_target)) {
    const taken = lines.some((l) => l.source === from.id && kindOf(l.target) === to.kind);
    if (taken) return { ok: false, reason: specs.find((s) => s.rule)?.rule ?? "This block already has a link of this kind." };
  }
  return { ok: true };
}

// What a line means and is good for, for the hover card.
export type LinkCard = { source: BlockKind; target: BlockKind; meaning: string; why: string };

const card = (e: EdgeSpec): LinkCard => ({ source: e.source, target: e.target, meaning: sentence(e.meaning), why: e.why ?? "" });

// sentence starts a manifest's meaning with a capital and ends it with a stop.
const sentence = (s: string) => {
  const t = s.trim();
  if (!t) return t;
  return t[0].toUpperCase() + t.slice(1) + (/[.!?]$/.test(t) ? "" : ".");
};

// linkCard is a line's card from the first cloud in play that draws it, or
// from any cloud in the catalog, or null.
export function linkCard(manifests: Manifest[], catalog: CatalogEntry[], source: BlockKind, target: BlockKind): LinkCard | null {
  const e = edgesOf(manifests, source, target)[0] ?? edgesOf(catalog.map((c) => c.manifest), source, target)[0];
  return e ? card(e) : null;
}

// allLinkCards is every line the clouds in play draw, once each, in the order
// the first cloud lists them.
export function allLinkCards(manifests: Manifest[]): LinkCard[] {
  const seen = new Set<string>();
  const out: LinkCard[] = [];
  for (const m of manifests) {
    for (const e of m.edges ?? []) {
      const key = `${e.source}>${e.target}`;
      if (seen.has(key)) continue;
      seen.add(key);
      out.push(card(e));
    }
  }
  return out;
}
