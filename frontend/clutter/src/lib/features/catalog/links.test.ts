import { describe, expect, it } from "vitest";
import { allLinkCards, linkCard, linkCheck, manifestsInPlay } from "./links";
import type { CatalogEntry, Manifest } from "./types";

const manifest = (connector: string, edges: Manifest["edges"]): Manifest => ({
  connector,
  title: connector,
  version: "1",
  blocks: [],
  edges,
  settings: null,
  display: { short_name: connector },
});

// Shaped like the two real manifests: AWS keeps a VM on one network, both keep
// a volume on one VM, and only AWS links a function to a bucket.
const aws = manifest("aws", [
  { source: "vm", target: "network", meaning: "the VM runs in the network", why: "Private talk.", single_target: true, rule: "A VM runs in one network." },
  { source: "volume", target: "vm", meaning: "the volume is attached to the VM", why: "More room.", single_target: true, rule: "A volume attaches to one VM." },
  { source: "function", target: "bucket", meaning: "the function can read the bucket", why: "Your code can read the files it needs." },
]);
const openstack = manifest("general-openstack", [
  { source: "vm", target: "network", meaning: "the VM is attached to the network", why: "Private talk." },
  { source: "volume", target: "vm", meaning: "the volume is attached to the VM", why: "More room.", single_target: true, rule: "A volume attaches to one VM." },
]);
const entry = (m: Manifest): CatalogEntry => ({ manifest: m, credentials: { kind: "stored", fields: null, linked_only: false }, has_linked_accounts: false });
const catalog = [entry(aws), entry(openstack)];

const titles: Record<string, string> = { vm: "VM", network: "Network", volume: "Volume", function: "Function", bucket: "Bucket", firewall: "Firewall" };
const titleOf = (k: string) => titles[k] ?? k;
const kinds: Record<string, string> = { web: "vm", db: "vm", disk: "volume", net: "network", fn: "function", files: "bucket", fw: "firewall" };
const kindOf = (id: string) => kinds[id];
const end = (id: string) => ({ id, kind: kinds[id] });

describe("manifestsInPlay", () => {
  it("keeps the clouds in play, or every cloud when none is", () => {
    expect(manifestsInPlay(catalog, ["general-openstack"]).map((m) => m.connector)).toEqual(["general-openstack"]);
    expect(manifestsInPlay(catalog, []).map((m) => m.connector)).toEqual(["aws", "general-openstack"]);
    expect(manifestsInPlay(catalog, ["gone"]).map((m) => m.connector)).toEqual(["aws", "general-openstack"]);
  });
});

describe("linkCheck", () => {
  const all = [aws, openstack];

  it("allows a line some cloud in play draws", () => {
    expect(linkCheck(all, end("disk"), end("web"), [], kindOf, titleOf)).toEqual({ ok: true });
    expect(linkCheck(all, end("fn"), end("files"), [], kindOf, titleOf)).toEqual({ ok: true });
  });

  it("refuses with one short line saying why", () => {
    expect(linkCheck(all, end("web"), end("web"), [], kindOf, titleOf)).toEqual({ ok: false, reason: "A block cannot link to itself." });
    expect(linkCheck(all, end("web"), end("net"), [], kindOf, titleOf)).toEqual({ ok: false, reason: "Drag the VM into the network box instead." });
    expect(linkCheck(all, end("files"), end("fw"), [], kindOf, titleOf)).toEqual({ ok: false, reason: "A Bucket cannot link to a Firewall." });
    expect(linkCheck(all, end("web"), end("disk"), [], kindOf, titleOf)).toEqual({
      ok: false,
      reason: "Draw it the other way, from the Volume to the VM.",
    });
    expect(linkCheck(all, end("disk"), end("web"), [{ source: "disk", target: "web" }], kindOf, titleOf)).toEqual({
      ok: false,
      reason: "These two are already linked.",
    });
  });

  it("refuses a second line the manifests' rule allows once, with the rule", () => {
    expect(linkCheck(all, end("disk"), end("db"), [{ source: "disk", target: "web" }], kindOf, titleOf)).toEqual({
      ok: false,
      reason: "A volume attaches to one VM.",
    });
  });

  it("reads only the clouds in play", () => {
    expect(linkCheck([openstack], end("fn"), end("files"), [], kindOf, titleOf)).toMatchObject({ ok: false });
  });
});

describe("link cards", () => {
  it("say what a line means and why, from the first cloud in play that draws it", () => {
    expect(linkCard([openstack], catalog, "vm", "network")).toEqual({
      source: "vm",
      target: "network",
      meaning: "The VM is attached to the network.",
      why: "Private talk.",
    });
    // Not in play, so it comes from the catalog.
    expect(linkCard([openstack], catalog, "function", "bucket")?.why).toBe("Your code can read the files it needs.");
    expect(linkCard([openstack], catalog, "bucket", "vm")).toBeNull();
  });

  it("list every line once, for How linking works", () => {
    expect(allLinkCards([aws, openstack]).map((c) => `${c.source}>${c.target}`)).toEqual(["vm>network", "volume>vm", "function>bucket"]);
  });
});
