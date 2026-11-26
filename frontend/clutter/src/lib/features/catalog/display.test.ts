import { describe, expect, it } from "vitest";
import { tintForId } from "../../tint";
import { blockLook, cloudLook, cloudLookFor, iconSrc, KIND_TINT, kindBadgeClass, kindIconClass, monogram } from "./display";
import type { CatalogEntry, Manifest } from "./types";

// Shaped like the AWS and OpenStack manifests: AWS has a service icon per
// block except the firewall; OpenStack names its services and keeps neutral icons.
const aws: Manifest = {
  connector: "aws",
  title: "Amazon Web Services",
  version: "1",
  blocks: [
    { kind: "vm", treatment: "as-is", title: "VM", display: { name: "EC2", icon: "ec2" } },
    { kind: "firewall", treatment: "as-is", title: "Firewall", display: { name: "Security group" } },
    { kind: "bucket", treatment: "as-is", title: "Bucket", display: { name: "S3", icon: "s3" } },
  ],
  edges: null,
  settings: null,
  display: { short_name: "AWS", logo: "logo" },
};

const openstack: Manifest = {
  connector: "general-openstack",
  title: "OpenStack",
  version: "1",
  blocks: [
    { kind: "vm", treatment: "as-is", title: "VM", display: { name: "Server (Nova)" } },
    { kind: "bucket", treatment: "removed", title: "Bucket" },
  ],
  edges: null,
  settings: null,
  display: { short_name: "OpenStack", logo: "logo" },
};

const entry = (manifest: Manifest): CatalogEntry => ({
  manifest,
  credentials: { kind: "stored", fields: null, linked_only: false },
  has_linked_accounts: false,
});

describe("iconSrc", () => {
  it("resolves a key to the connector's folder", () => {
    expect(iconSrc("aws", "ec2")).toBe("/clouds/aws/ec2.svg");
    expect(iconSrc("general-openstack", "logo")).toBe("/clouds/general-openstack/logo.svg");
  });

  it("refuses an empty key and anything that is not a plain key", () => {
    expect(iconSrc("aws", "")).toBeNull();
    expect(iconSrc("aws", undefined)).toBeNull();
    expect(iconSrc("aws", "../mock/logo")).toBeNull();
    expect(iconSrc("aws", "ec2.svg")).toBeNull();
    expect(iconSrc("aws", "EC2")).toBeNull();
    expect(iconSrc("../x", "logo")).toBeNull();
  });
});

describe("cloudLook", () => {
  it("names a cloud by its short name and logo", () => {
    expect(cloudLook(aws)).toEqual({ connector: "aws", title: "Amazon Web Services", shortName: "AWS", logoSrc: "/clouds/aws/logo.svg" });
  });

  it("falls back to the title, and to the id for a cloud the catalog no longer offers", () => {
    expect(cloudLook({ ...aws, display: { short_name: "" } })).toMatchObject({ shortName: "Amazon Web Services", logoSrc: null });
    expect(cloudLookFor([entry(aws)], "gone")).toEqual({ connector: "gone", title: "gone", shortName: "gone", logoSrc: null });
    expect(cloudLookFor([entry(aws), entry(openstack)], "general-openstack").shortName).toBe("OpenStack");
  });
});

describe("blockLook", () => {
  it("gives the cloud's service name and icon", () => {
    expect(blockLook(aws, "vm", "VM")).toEqual({ name: "EC2", iconSrc: "/clouds/aws/ec2.svg", offered: true });
    expect(blockLook(openstack, "vm", "VM")).toEqual({ name: "Server (Nova)", iconSrc: null, offered: true });
  });

  it("keeps the neutral icon when the cloud names no icon", () => {
    expect(blockLook(aws, "firewall", "Firewall")).toEqual({ name: "Security group", iconSrc: null, offered: true });
  });

  it("marks a removed or missing block as not offered, under the block's own title", () => {
    expect(blockLook(openstack, "bucket", "Bucket")).toEqual({ name: "Bucket", iconSrc: null, offered: false });
    expect(blockLook(aws, "table", "Table")).toEqual({ name: "Table", iconSrc: null, offered: false });
  });
});

describe("monogram", () => {
  it("keeps a short name of three characters or fewer, else its first letter", () => {
    expect(monogram("AWS")).toBe("AWS");
    expect(monogram("OpenStack")).toBe("O");
    expect(monogram(" mock ")).toBe("M");
  });
});

describe("block kind colours", () => {
  it("gives each of B's eight kinds its own tint", () => {
    const kinds = ["vm", "network", "firewall", "volume", "bucket", "function", "table", "http_api"];
    expect(new Set(kinds.map((k) => KIND_TINT[k])).size).toBe(8);
    expect(kindBadgeClass("vm")).toBe("icon-tile bg-tint-blue border-tint-blue-edge text-tint-blue-fg");
    expect(kindIconClass("network")).toBe("text-tint-violet-fg");
  });

  it("shows a kind a cloud adds on its own as neutral", () => {
    expect(kindBadgeClass("queue")).toBe("icon-tile bg-muted text-foreground");
    expect(kindIconClass("queue")).toBe("text-muted-foreground");
  });
});

describe("tintForId", () => {
  it("picks the same readable tint for the same id every time", () => {
    const id = "e6415f8a-1fd0-4b3b-9094-3931ad461366";
    expect(tintForId(id)).toBe(tintForId(id));
    expect(["violet", "grass", "pink"]).toContain(tintForId(id));
    expect(new Set(["a", "b", "c", "d", "e", "f"].map(tintForId)).size).toBeGreaterThan(1);
  });
});
