import { describe, expect, it } from "vitest";
import type { CatalogEntry } from "../catalog/types";
import { cloudSummary, connectionStatusLabel, credentialHelp, offeredClouds, preferredCloud } from "./display";

describe("connectionStatusLabel", () => {
  it("says Verified or Not working, one word set for every screen", () => {
    expect(connectionStatusLabel("working")).toBe("Verified");
    expect(connectionStatusLabel("failed")).toBe("Not working");
  });
});

const entry = (connector: string, credentials: Partial<CatalogEntry["credentials"]> = {}, hasLinked = false): CatalogEntry => ({
  manifest: { connector, title: connector, version: "1", blocks: null, edges: null, settings: null, display: { short_name: connector } },
  credentials: { kind: "stored", fields: null, linked_only: false, ...credentials },
  has_linked_accounts: hasLinked,
});

describe("offeredClouds", () => {
  const catalog = [entry("aws"), entry("general-openstack"), entry("mock")];

  it("offers the test cloud only in a development build", () => {
    expect(offeredClouds(catalog, true).map((e) => e.manifest.connector)).toEqual(["aws", "general-openstack", "mock"]);
    expect(offeredClouds(catalog, false).map((e) => e.manifest.connector)).toEqual(["aws", "general-openstack"]);
  });
});

describe("cloudSummary", () => {
  it("uses the cloud's own line, else the credential spec's", () => {
    expect(cloudSummary(entry("mock"))).toBe("Test cloud (builds nothing, local only)");
    expect(cloudSummary(entry("other", { linked_only: true }))).toBe("Link your account, or use one you linked");
    expect(cloudSummary(entry("other", {}, true))).toBe("Use an account you linked, or enter its details");
    expect(cloudSummary(entry("other", { kind: "borrowed" }))).toBe("Borrowed access");
    expect(cloudSummary(entry("other"))).toBe("Credentials stored encrypted");
  });
});

describe("credentialHelp", () => {
  it("never shows the test cloud's token", () => {
    const help = credentialHelp("mock", "token", "Any mock token. Only mock-valid-token works.");
    expect(help).not.toContain("mock-valid-token");
  });

  it("keeps the manifest's help for every other field", () => {
    expect(credentialHelp("general-openstack", "auth_url", "Ends in /v3.")).toBe("Ends in /v3.");
    expect(credentialHelp("mock", "other", undefined)).toBeUndefined();
  });
});

describe("preferredCloud", () => {
  const offered = [entry("aws"), entry("general-openstack")];

  it("takes the first wanted cloud the build offers", () => {
    expect(preferredCloud(offered, "general-openstack", "aws")).toBe("general-openstack");
    expect(preferredCloud(offered, undefined, "aws")).toBe("aws");
    expect(preferredCloud(offered, "mock", null)).toBeNull();
  });
});
