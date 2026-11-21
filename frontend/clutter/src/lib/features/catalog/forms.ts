// The manifest-to-form mapping. B's editor builds its palette, block forms
// and deployment settings from the catalog only (links.ts reads the edges;
// the manifests
// are the one owner of blocks, fields, defaults and allowed values). The value
// rules here mirror connectors.CheckValue in backend/adapter/connectors/manifest.go;
// B checks again before anything runs.

import { cloudLook, type CloudLook } from "./display";
import { blockValueError } from "./fieldRules";
import { MAX_STRING_FIELD, MAX_TEXT_FIELD, STANDARD_BLOCKS } from "./types";
import type { BlockKind, CatalogEntry, FieldSpec, Manifest } from "./types";

export type PaletteBlock = {
  kind: BlockKind;
  title: string;
  // What it is for, from the first cloud that says; "" when none does.
  summary: string;
  // The clouds that can build it.
  clouds: CloudLook[];
};

// manifestOffers reports whether a cloud can build a kind of block.
export const manifestOffers = (m: Manifest, kind: BlockKind) =>
  (m.blocks ?? []).some((b) => b.kind === kind && b.treatment !== "removed");

// entryFor is a connector's catalog entry, or null when this server no
// longer offers the connector.
export const entryFor = (catalog: CatalogEntry[], connector: string | null | undefined): CatalogEntry | null =>
  connector ? (catalog.find((e) => e.manifest.connector === connector) ?? null) : null;

// cloudOffers reports whether a connector in the catalog can build a kind of block.
export const cloudOffers = (catalog: CatalogEntry[], connector: string, kind: BlockKind) => {
  const entry = entryFor(catalog, connector);
  return !!entry && manifestOffers(entry.manifest, kind);
};

// paletteBlocks lists every block at least one cloud can build: the standard
// blocks in their fixed order, then any cloud's own blocks by title.
export function paletteBlocks(catalog: CatalogEntry[]): PaletteBlock[] {
  const byKind = new Map<BlockKind, PaletteBlock>();
  for (const { manifest } of catalog) {
    for (const b of manifest.blocks ?? []) {
      if (b.treatment === "removed") continue;
      const cur = byKind.get(b.kind);
      if (cur) {
        if (!cur.clouds.some((c) => c.connector === manifest.connector)) cur.clouds.push(cloudLook(manifest));
        if (!cur.summary && b.summary) cur.summary = b.summary;
      } else {
        byKind.set(b.kind, { kind: b.kind, title: b.title, summary: b.summary ?? "", clouds: [cloudLook(manifest)] });
      }
    }
  }
  const standard = STANDARD_BLOCKS.flatMap((k) => (byKind.has(k) ? [byKind.get(k)!] : []));
  const own = [...byKind.values()].filter((b) => !STANDARD_BLOCKS.includes(b.kind)).sort((a, b) => a.title.localeCompare(b.title));
  return [...standard, ...own];
}

// blockTitle names a kind as the manifests do, or by its kind when none offers it.
export function blockTitle(catalog: CatalogEntry[], kind: BlockKind): string {
  for (const { manifest } of catalog) {
    const b = (manifest.blocks ?? []).find((x) => x.kind === kind);
    if (b?.title) return b.title;
  }
  return kind;
}

// blockFields returns a block's form fields: every field any cloud that builds
// it declares, first declaration first. An enum keeps only the values every
// declaring cloud allows, so a value picked here passes each of them.
export function blockFields(catalog: CatalogEntry[], kind: BlockKind): FieldSpec[] {
  const out: FieldSpec[] = [];
  for (const { manifest } of catalog) {
    const b = (manifest.blocks ?? []).find((x) => x.kind === kind && x.treatment !== "removed");
    for (const f of b?.fields ?? []) {
      const i = out.findIndex((x) => x.name === f.name);
      if (i < 0) {
        out.push({ ...f, allowed: f.allowed ? [...f.allowed] : undefined });
        continue;
      }
      const cur = out[i];
      if (cur.type === "enum" && f.type === "enum" && cur.allowed && f.allowed) {
        const both = cur.allowed.filter((v) => f.allowed!.includes(v));
        if (both.length > 0) out[i] = { ...cur, allowed: both };
      }
    }
  }
  return out;
}

// The network field that gives private servers internet access
// (connectors' standard network field). It matters only for a network with
// a VM in it (a private one on AWS, any on OpenStack), so it shows only then.
export const PRIVATE_INTERNET_FIELD = "private_internet";

// fieldShown reports whether a block's field is worth showing: every field
// is, except internet access for private servers on a network with no VM.
export const fieldShown = (f: FieldSpec, ctx: { hasVM: boolean }) => f.name !== PRIVATE_INTERNET_FIELD || ctx.hasVM;

// connectedBuilders are the clouds that can build a kind and that the
// organization has a connection to, with each cloud's service name and its
// help lines for the block.
export function connectedBuilders(
  catalog: CatalogEntry[],
  kind: BlockKind,
  connected: ReadonlySet<string>,
): { cloud: CloudLook; service: string; help: string[] }[] {
  return catalog.flatMap(({ manifest }) => {
    const b = (manifest.blocks ?? []).find((x) => x.kind === kind && x.treatment !== "removed");
    if (!b || !connected.has(manifest.connector)) return [];
    return [{ cloud: cloudLook(manifest), service: b.display?.name || b.title, help: b.help ?? [] }];
  });
}

// settingsFor keeps the deployment settings that are about the diagram's
// blocks: a function runtime only when the diagram has a function. A setting
// that names no kind is about every deployment.
export function settingsFor(settings: FieldSpec[] | null | undefined, kinds: Iterable<BlockKind>): FieldSpec[] {
  const present = new Set(kinds);
  return (settings ?? []).filter((f) => !f.applies_to || f.applies_to.length === 0 || f.applies_to.some((k) => present.has(k)));
}

// defaultValues holds each field's default, leaving out fields with none.
export function defaultValues(fields: FieldSpec[]): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const f of fields) {
    if (f.default !== undefined && f.default !== null) out[f.name] = f.default;
  }
  return out;
}

// What a field renders as.
export type FieldControl =
  | { control: "select"; options: string[] }
  | { control: "checkbox" }
  | { control: "number" }
  | { control: "text" }
  | { control: "textarea" };

export function fieldControl(f: FieldSpec): FieldControl {
  switch (f.type) {
    case "enum":
      return { control: "select", options: f.allowed ?? [] };
    case "bool":
      return { control: "checkbox" };
    case "int":
      return { control: "number" };
    case "text":
      return { control: "textarea" };
    default:
      return { control: "text" };
  }
}

// fieldLabel splits a manifest title that carries its own example, such as
// "Allowed incoming traffic (for example tcp/22,tcp/443)", into the label
// and a hint the form shows under the field ("For example tcp/22,tcp/443."),
// so the label stays short. Any other title is the label as it is.
export function fieldLabel(title: string): { label: string; hint: string | null } {
  const m = title.match(/^(.+?)\s*\((for example\b[^)]*)\)\s*$/i);
  if (!m) return { label: title, hint: null };
  const example = m[2].trim();
  return { label: m[1], hint: `${example[0].toUpperCase()}${example.slice(1)}${/[.!?]$/.test(example) ? "" : "."}` };
}

// textBytes is a text value's length as B counts it: UTF-8 bytes.
export const textBytes = (s: string) => new TextEncoder().encode(s).length;

// parseInput turns what a control holds into the value the document stores.
// An empty text or number box means "not set" (undefined), so the default or
// the required check applies.
export function parseInput(f: FieldSpec, raw: string | boolean): unknown {
  if (f.type === "bool") return raw === true || raw === "true";
  const s = String(raw);
  // A text area keeps what is typed, leading blank lines included; only an
  // empty one is unset. B treats a blank script as none.
  if (f.type === "text") return s === "" ? undefined : s;
  if (s.trim() === "") return undefined;
  if (f.type === "int") {
    const n = Number(s);
    return Number.isFinite(n) ? n : s;
  }
  return s;
}

// fieldError returns why value is not valid for f, or null. It mirrors
// connectors.CheckValue, plus the required check B's capability check makes.
export function fieldError(f: FieldSpec, value: unknown): string | null {
  if (value === undefined || value === null || value === "") {
    return f.required && (f.default === undefined || f.default === null) ? `${f.title} is required.` : null;
  }
  switch (f.type) {
    case "string":
      if (typeof value !== "string") return `${f.title} must be text.`;
      if (value.length > MAX_STRING_FIELD) return `${f.title} must be at most ${MAX_STRING_FIELD} characters.`;
      return null;
    case "text":
      if (typeof value !== "string") return `${f.title} must be text.`;
      if (textBytes(value) > MAX_TEXT_FIELD) return `${f.title} must be at most ${MAX_TEXT_FIELD / 1024} KB.`;
      return null;
    case "int":
      if (typeof value !== "number" || !Number.isInteger(value) || Math.abs(value) > 2 ** 53) {
        return `${f.title} must be a whole number.`;
      }
      return null;
    case "bool":
      return typeof value === "boolean" ? null : `${f.title} must be on or off.`;
    case "enum":
      return typeof value === "string" && (f.allowed ?? []).includes(value) ? null : `${f.title} must be one of the listed values.`;
    default:
      return `${f.title} has a type this page does not know.`;
  }
}

// valuesErrors checks every field of a form, in field order.
export function valuesErrors(fields: FieldSpec[], values: Record<string, unknown>): { field: string; message: string }[] {
  return fields.flatMap((f) => {
    const msg = fieldError(f, values[f.name]);
    return msg ? [{ field: f.name, message: msg }] : [];
  });
}

// blockFieldError is fieldError plus the grammar of a block's portable field
// (a firewall's allowed traffic, an address range; fieldRules.ts).
export const blockFieldError = (kind: BlockKind, f: FieldSpec, value: unknown): string | null =>
  fieldError(f, value) ?? blockValueError(kind, f.name, value);

// blockValuesErrors checks every field of a block, in field order.
export function blockValuesErrors(kind: BlockKind, fields: FieldSpec[], values: Record<string, unknown>): { field: string; message: string }[] {
  return fields.flatMap((f) => {
    const msg = blockFieldError(kind, f, values[f.name]);
    return msg ? [{ field: f.name, message: msg }] : [];
  });
}

// cleanValues drops unset values and names no field declares, so a saved
// document or settings object holds only what the manifest knows.
export function cleanValues(fields: FieldSpec[], values: Record<string, unknown>): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const f of fields) {
    const v = values[f.name];
    if (v !== undefined && v !== null && v !== "") out[f.name] = v;
  }
  return out;
}
