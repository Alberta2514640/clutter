import { describe, expect, it } from "vitest";
import { APP_CRED_ID, APP_CRED_SECRET, AUTH_URL, fitsCloudsYaml, readCloudsFiles, splitCloudsFiles, type CloudsEntry } from "./cloudsYaml";

// The shape Horizon downloads for an application credential.
const appCred = `
# This is a clouds.yaml file, which can be used by OpenStack tools as a source
# of configuration on how to connect to a cloud.
clouds:
  openstack:
    auth:
      auth_url: https://cloud.example.com:5000
      application_credential_id: "21f1e0a8c6d24c84a1f7c1f7f8f01a6b"
      application_credential_secret: "0123456789"
    region_name: "RegionOne"
    interface: "public"
    identity_api_version: 3
    auth_type: "v3applicationcredential"
`;

// The shape Horizon downloads from the user menu: a password entry.
const password = `
clouds:
  openstack:
    auth:
      auth_url: https://cloud.example.com:5000/v3
      username: "demo"
      project_id: 7a1f
      project_name: "demo"
      user_domain_name: "Default"
    region_name: "RegionOne"
    interface: "public"
    identity_api_version: 3
`;

const only = (text: string, secure?: string): CloudsEntry => {
  const r = readCloudsFiles(text, secure);
  if (!r.ok) throw new Error(r.message);
  expect(r.entries).toHaveLength(1);
  return r.entries[0];
};

describe("readCloudsFiles", () => {
  it("fills the form from an application credential entry", () => {
    const e = only(appCred);
    expect(e).toEqual({
      name: "openstack",
      ok: true,
      region: "RegionOne",
      missing: [],
      values: {
        [AUTH_URL]: "https://cloud.example.com:5000/v3",
        [APP_CRED_ID]: "21f1e0a8c6d24c84a1f7c1f7f8f01a6b",
        // Digits stay text, exactly as written.
        [APP_CRED_SECRET]: "0123456789",
      },
    });
  });

  it("keeps an auth URL that already ends in /v3, without a trailing slash", () => {
    const e = only(appCred.replace("https://cloud.example.com:5000", "http://10.0.0.5/identity/v3/"));
    expect(e.ok && e.values[AUTH_URL]).toBe("http://10.0.0.5/identity/v3");
  });

  it("refuses a password entry and says how to make an application credential", () => {
    const e = only(password);
    expect(e.ok).toBe(false);
    if (e.ok) return;
    expect(e.message).toContain("user name and password");
    expect(e.message).toContain("Identity, then Application Credentials");
    expect(e.message).toContain("Create Application Credential");
  });

  it("refuses an explicit password auth type even with a password in secure.yaml", () => {
    const e = only(
      password.replace("identity_api_version: 3", "identity_api_version: 3\n    auth_type: password"),
      `clouds:\n  openstack:\n    auth:\n      password: hunter2\n`,
    );
    expect(e.ok).toBe(false);
  });

  it("refuses another sign-in method", () => {
    const e = only(`clouds:\n  sso:\n    auth_type: v3oidcpassword\n    auth:\n      auth_url: https://k.example.com/v3\n`);
    expect(e.ok === false && e.message).toContain("sign-in method Clutter does not take");
  });

  it("refuses a credential named but without its ID", () => {
    const e = only(`clouds:\n  c:\n    auth_type: v3applicationcredential\n    auth:\n      auth_url: https://k.example.com/v3\n      application_credential_name: ci\n      username: demo\n`);
    expect(e.ok === false && e.message).toContain("instead of giving its ID");
  });

  it("refuses Identity v2", () => {
    const e = only(appCred.replace("https://cloud.example.com:5000", "https://cloud.example.com:5000/v2.0"));
    expect(e.ok === false && e.message).toContain("Identity API v2");
  });

  it("lists several clouds, each ready or refused", () => {
    const text = `
clouds:
  lab:
    auth_type: v3applicationcredential
    auth:
      auth_url: https://lab.example.com:5000
      application_credential_id: lab-id
      application_credential_secret: lab-secret
    region_name: RegionOne
  prod:
    auth_type: v3applicationcredential
    auth:
      auth_url: https://prod.example.com/identity
      application_credential_id: prod-id
      application_credential_secret: prod-secret
    regions:
      - name: ca-west-1
      - name: ca-east-1
  old:
    auth:
      auth_url: https://old.example.com/v3
      username: me
      password: pw
`;
    const r = readCloudsFiles(text);
    expect(r.ok).toBe(true);
    if (!r.ok) return;
    expect(r.entries.map((e) => [e.name, e.ok])).toEqual([
      ["lab", true],
      ["prod", true],
      ["old", false],
    ]);
    const prod = r.entries[1];
    expect(prod.ok && prod.values[AUTH_URL]).toBe("https://prod.example.com/identity/v3");
    expect(prod.ok && prod.region).toBe("ca-west-1");
  });

  it("names the fields an entry lacks, and takes the secret from secure.yaml", () => {
    const noSecret = appCred.replace(/\n\s*application_credential_secret:.*\n/, "\n").replace(/\n\s*region_name:.*\n/, "\n");
    const e = only(noSecret);
    expect(e.ok && e.missing).toEqual(["application credential secret", "region"]);
    const withSecure = only(noSecret, `clouds:\n  openstack:\n    auth:\n      application_credential_secret: from-secure\n    region_name: RegionTwo\n`);
    expect(withSecure.ok && withSecure.values[APP_CRED_SECRET]).toBe("from-secure");
    expect(withSecure.ok && withSecure.region).toBe("RegionTwo");
    expect(withSecure.ok && withSecure.missing).toEqual([]);
  });

  it("names a missing auth URL and ID", () => {
    const e = only(`clouds:\n  c:\n    auth_type: v3applicationcredential\n    auth:\n      application_credential_secret: s\n    region_name: RegionOne\n`);
    expect(e.ok && e.missing).toEqual(["auth URL", "application credential ID"]);
  });

  it("refuses a file that is not a clouds file, without quoting it", () => {
    const bad = readCloudsFiles("clouds: [\n  application_credential_secret: s3cr3t");
    expect(bad.ok).toBe(false);
    expect(!bad.ok && bad.message).toBe("The clouds.yaml file is not valid YAML.");
    expect(readCloudsFiles("name: x\n")).toEqual({ ok: false, message: expect.stringContaining("no clouds section") });
    expect(readCloudsFiles("clouds: {}\n")).toEqual({ ok: false, message: "The clouds.yaml file lists no clouds." });
    expect(readCloudsFiles("a".repeat(70 * 1024))).toEqual({ ok: false, message: expect.stringContaining("too large") });
    expect(readCloudsFiles(appCred, "clouds: [")).toEqual({ ok: false, message: "The secure.yaml file is not valid YAML." });
  });

  it("refuses repeated keys and alias bombs", () => {
    expect(readCloudsFiles("clouds:\n  a: {}\n  a: {}\n").ok).toBe(false);
    const bomb = ["a: &a [x,x,x,x,x,x,x,x,x]"];
    for (let i = 0; i < 12; i++) bomb.push(`${String.fromCharCode(98 + i)}: &${String.fromCharCode(98 + i)} [*${String.fromCharCode(97 + i)},*${String.fromCharCode(97 + i)},*${String.fromCharCode(97 + i)}]`);
    expect(readCloudsFiles(`${bomb.join("\n")}\nclouds: {}\n`).ok).toBe(false);
  });
});

describe("splitCloudsFiles and fitsCloudsYaml", () => {
  it("tells clouds.yaml from secure.yaml by name", () => {
    const r = splitCloudsFiles([{ name: "secure.yaml" }, { name: "clouds.yaml" }]);
    expect(r).toEqual({ ok: true, clouds: { name: "clouds.yaml" }, secure: { name: "secure.yaml" } });
    expect(splitCloudsFiles([{ name: "openstack-clouds.yml" }])).toEqual({ ok: true, clouds: { name: "openstack-clouds.yml" }, secure: null });
    expect(splitCloudsFiles([{ name: "a.yaml" }, { name: "b.yaml" }]).ok).toBe(false);
    expect(splitCloudsFiles([{ name: "secure.yaml" }]).ok).toBe(false);
  });

  it("offers the upload only to a credential spec with the clouds.yaml fields", () => {
    expect(fitsCloudsYaml([AUTH_URL, APP_CRED_ID, APP_CRED_SECRET])).toBe(true);
    expect(fitsCloudsYaml(["token"])).toBe(false);
  });
});
