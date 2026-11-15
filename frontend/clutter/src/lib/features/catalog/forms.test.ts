import { describe, expect, it } from "vitest";
import {
  blockFields,
  blockTitle,
  cleanValues,
  defaultValues,
  fieldControl,
  fieldError,
  paletteBlocks,
  parseInput,
  valuesErrors,
} from "./forms";
import type { CatalogEntry, FieldSpec } from "./types";

const size: FieldSpec = { name: "size", title: "Size", type: "enum", required: true, default: "small", allowed: ["small", "medium", "large"] };
const sizeGB: FieldSpec = { name: "size_gb", title: "Size in GB", type: "int", required: true, default: 10 };
const cidr: FieldSpec = { name: "cidr", title: "Address range (CIDR)", type: "string", required: true, default: "10.0.0.0/24" };
const isPublic: FieldSpec = { name: "public", title: "Reachable from the internet", type: "bool", required: false, default: false };

// Shaped like the catalog route's answer: an OpenStack-like cloud without
// buckets or functions, and an AWS-like cloud with them and a narrower size.
const catalog: CatalogEntry[] = [
  {
    has_linked_accounts: false,
    credentials: { kind: "stored", fields: [{ name: "token", label: "Token", secret: true, required: true }], linked_only: false },
    manifest: {
      connector: "general-openstack",
      title: "OpenStack",
      version: "1",
      blocks: [
        { kind: "vm", treatment: "as-is", title: "VM", summary: "A server that runs your app", fields: [size, isPublic] },
        { kind: "network", treatment: "as-is", title: "Network", fields: [cidr] },
        { kind: "volume", treatment: "as-is", title: "Volume", fields: [sizeGB] },
        { kind: "bucket", treatment: "removed", title: "Bucket" },
        { kind: "function", treatment: "removed", title: "Function" },
      ],
      edges: [
        { source: "vm", target: "network", meaning: "the VM is attached to the network" },
        { source: "volume", target: "vm", meaning: "the volume is attached to the VM" },
      ],
      settings: [{ name: "image", title: "Image", type: "string", required: true, default: "cirros" }],
      display: { short_name: "OpenStack", logo: "logo" },
    },
  },
  {
    has_linked_accounts: true,
    credentials: { kind: "borrowed", fields: null, linked_only: true },
    manifest: {
      connector: "aws",
      title: "Amazon Web Services",
      version: "1",
      regions: ["ca-central-1"],
      blocks: [
        { kind: "vm", treatment: "as-is", title: "VM", fields: [{ ...size, allowed: ["small", "medium"] }] },
        { kind: "bucket", treatment: "as-is", title: "Bucket", fields: [{ name: "versioning", title: "Keep old versions", type: "bool", required: false }] },
        { kind: "function", treatment: "as-is", title: "Function" },
        { kind: "queue", treatment: "new", title: "Queue" },
      ],
      edges: [{ source: "function", target: "bucket", meaning: "the function may read the bucket" }],
      settings: null,
      display: { short_name: "AWS", logo: "logo" },
    },
  },
];

describe("paletteBlocks", () => {
  it("lists every block some cloud builds, standard blocks first in their fixed order, with the clouds that build it", () => {
    const palette = paletteBlocks(catalog);
    expect(palette.map((b) => b.kind)).toEqual(["vm", "network", "volume", "bucket", "function", "queue"]);
    expect(palette.find((b) => b.kind === "vm")?.clouds.map((c) => c.title)).toEqual(["OpenStack", "Amazon Web Services"]);
    // The summary comes from the first cloud that has one.
    expect(palette.find((b) => b.kind === "vm")?.summary).toBe("A server that runs your app");
    expect(palette.find((b) => b.kind === "network")?.summary).toBe("");
    expect(palette.find((b) => b.kind === "bucket")?.clouds).toEqual([
      { connector: "aws", title: "Amazon Web Services", shortName: "AWS", logoSrc: "/clouds/aws/logo.svg" },
    ]);
  });

  it("is empty for an empty catalog, and names an unknown kind by itself", () => {
    expect(paletteBlocks([])).toEqual([]);
    expect(blockTitle(catalog, "volume")).toBe("Volume");
    expect(blockTitle(catalog, "mystery")).toBe("mystery");
  });
});

describe("blockFields", () => {
  it("joins the fields every cloud declares, keeping only enum values all of them allow", () => {
    const fields = blockFields(catalog, "vm");
    expect(fields.map((f) => f.name)).toEqual(["size", "public"]);
    expect(fields[0].allowed).toEqual(["small", "medium"]);
  });

  it("ignores a cloud that removes the block, and does not change the catalog", () => {
    expect(blockFields(catalog, "bucket").map((f) => f.name)).toEqual(["versioning"]);
    blockFields(catalog, "vm")[0].allowed!.push("huge");
    expect(catalog[0].manifest.blocks![0].fields![0].allowed).toEqual(["small", "medium", "large"]);
  });

  it("gives a new block its defaults", () => {
    expect(defaultValues(blockFields(catalog, "vm"))).toEqual({ size: "small", public: false });
    expect(defaultValues(blockFields(catalog, "function"))).toEqual({});
  });
});

describe("field controls and values", () => {
  it("maps each field type to a control", () => {
    expect(fieldControl(size)).toEqual({ control: "select", options: ["small", "medium", "large"] });
    expect(fieldControl(isPublic)).toEqual({ control: "checkbox" });
    expect(fieldControl(sizeGB)).toEqual({ control: "number" });
    expect(fieldControl(cidr)).toEqual({ control: "text" });
  });

  it("parses what a control holds", () => {
    expect(parseInput(sizeGB, "20")).toBe(20);
    expect(parseInput(sizeGB, "")).toBeUndefined();
    expect(parseInput(isPublic, true)).toBe(true);
    expect(parseInput(cidr, "10.1.0.0/16")).toBe("10.1.0.0/16");
    expect(parseInput(cidr, "  ")).toBeUndefined();
  });

  it("checks values the way B's manifest does", () => {
    expect(fieldError(size, "medium")).toBeNull();
    expect(fieldError(size, "huge")).toBe("Size must be one of the listed values.");
    expect(fieldError(sizeGB, 2.5)).toBe("Size in GB must be a whole number.");
    expect(fieldError(sizeGB, "abc")).toBe("Size in GB must be a whole number.");
    expect(fieldError(cidr, "x".repeat(257))).toBe("Address range (CIDR) must be at most 256 characters.");
    expect(fieldError(isPublic, "yes")).toBe("Reachable from the internet must be on or off.");
    // Unset is fine when a default fills it, and refused when required with none.
    expect(fieldError(sizeGB, undefined)).toBeNull();
    expect(fieldError({ ...sizeGB, default: undefined }, undefined)).toBe("Size in GB is required.");
  });

  it("draws a text field as a text area and caps it at 16 KB of UTF-8, as B does", () => {
    const script: FieldSpec = { name: "startup_script", title: "Startup script", type: "text", required: false };
    expect(fieldControl(script)).toEqual({ control: "textarea" });
    const multiLine = "#!/bin/bash\napt-get install -y nginx\n";
    expect(parseInput(script, multiLine)).toBe(multiLine);
    // A text area keeps what is typed while it is typed; only empty is unset.
    expect(parseInput(script, "\n")).toBe("\n");
    expect(parseInput(script, "")).toBeUndefined();
    expect(fieldError(script, undefined)).toBeNull();
    expect(fieldError(script, "x".repeat(16384))).toBeNull();
    expect(fieldError(script, "x".repeat(16385))).toBe("Startup script must be at most 16 KB.");
    // 8192 two-byte characters are 16384 bytes; one more is over.
    expect(fieldError(script, "é".repeat(8192))).toBeNull();
    expect(fieldError(script, "é".repeat(8193))).toBe("Startup script must be at most 16 KB.");
    expect(fieldError(script, 3)).toBe("Startup script must be text.");
  });

  it("lists every field's error and keeps only declared, set values", () => {
    expect(valuesErrors([size, sizeGB], { size: "huge", size_gb: 1.5 })).toEqual([
      { field: "size", message: "Size must be one of the listed values." },
      { field: "size_gb", message: "Size in GB must be a whole number." },
    ]);
    expect(cleanValues([size, cidr], { size: "small", cidr: "", stray: 1 })).toEqual({ size: "small" });
  });
});
