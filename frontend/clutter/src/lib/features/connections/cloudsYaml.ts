// Reads the clouds.yaml (and optional secure.yaml) an OpenStack dashboard
// downloads, in the browser, and turns one cloud entry into the values of the
// connection form: auth URL, application credential ID and secret, and the
// region. Nothing here sends or stores the files; the form sends only the
// fields it already sends.
//
// Only application credentials are taken. A password entry is refused with
// how to make an application credential, because Clutter never holds a cloud
// password (backend/adapter/connectors/generalopenstack).

import { parse } from "yaml";

// A dashboard's clouds.yaml is well under a kilobyte.
export const MAX_CLOUDS_FILE = 64 * 1024;

// connectors.MaxCredentialValue
const MAX_VALUE = 8192;

// The credential fields an entry fills, named as clouds.yaml names its auth
// keys. A credential spec with these fields can take an upload.
export const AUTH_URL = "auth_url";
export const APP_CRED_ID = "application_credential_id";
export const APP_CRED_SECRET = "application_credential_secret";

export const APP_CRED_HELP =
  "In your cloud's dashboard, open Identity, then Application Credentials, and choose Create Application Credential. Download the clouds.yaml it offers and add it here.";

// fitsCloudsYaml reports whether a credential spec's fields are a clouds.yaml
// entry's application credential fields, so the upload can fill them.
export const fitsCloudsYaml = (fieldNames: string[]) =>
  [AUTH_URL, APP_CRED_ID, APP_CRED_SECRET].every((n) => fieldNames.includes(n));

export type CloudsEntry =
  | {
      name: string;
      ok: true;
      // Keyed by credential field name.
      values: Record<string, string>;
      region: string;
      // What the entry lacks, in words, for example "application credential secret".
      missing: string[];
    }
  | { name: string; ok: false; message: string };

export type CloudsFile = { ok: true; entries: CloudsEntry[] } | { ok: false; message: string };

type Obj = Record<string, unknown>;

const isObj = (v: unknown): v is Obj => typeof v === "object" && v !== null && !Array.isArray(v);

const own = (o: Obj, key: string): unknown => (Object.prototype.hasOwnProperty.call(o, key) ? o[key] : undefined);

// str returns a trimmed text value, or "" for anything else.
const str = (v: unknown): string => (typeof v === "string" ? v.trim() : "");

// readYaml parses one file. The failsafe schema reads every value as text, so
// an ID or a secret made of digits keeps its exact characters. logLevel
// "error" throws on any error (and logs no warning); "silent" would drop the
// errors and return a half-read file. The parser's own error is not shown: it
// quotes the file, which holds the secret.
function readYaml(text: string, file: string): { ok: true; doc: Obj } | { ok: false; message: string } {
  if (text.length > MAX_CLOUDS_FILE) return { ok: false, message: `The ${file} file is too large to be a clouds file.` };
  let doc: unknown;
  try {
    doc = parse(text, { schema: "failsafe", maxAliasCount: 10, uniqueKeys: true, prettyErrors: false, logLevel: "error" });
  } catch {
    return { ok: false, message: `The ${file} file is not valid YAML.` };
  }
  if (!isObj(doc) || !isObj(own(doc, "clouds"))) {
    return { ok: false, message: `The ${file} file has no clouds section. Use the clouds.yaml your cloud's dashboard downloads.` };
  }
  return { ok: true, doc };
}

// normalizeAuthUrl makes a clouds.yaml auth URL the Identity v3 address the
// form takes. Dashboards often write it without the version
// (https://cloud.example.com:5000), so /v3 is added.
function normalizeAuthUrl(raw: string, apiVersion: string): { ok: true; url: string } | { ok: false; message: string } {
  let u: URL;
  try {
    u = new URL(raw);
  } catch {
    return { ok: false, message: "The auth URL in this entry is not a web address." };
  }
  if (u.protocol !== "https:" && u.protocol !== "http:") {
    return { ok: false, message: "The auth URL in this entry is not a web address." };
  }
  const base = raw.replace(/\/+$/, "");
  const last = base.slice(base.lastIndexOf("/") + 1);
  if (/^v2(\.0)?$/i.test(last) || /^2(\.0)?$/.test(apiVersion)) {
    return { ok: false, message: "This entry uses the old Identity API v2. Clutter needs Identity v3; ask your cloud for its v3 address." };
  }
  return { ok: true, url: last === "v3" ? base : `${base}/v3` };
}

function entryRegion(entry: Obj): string {
  const r = str(own(entry, "region_name"));
  if (r) return r;
  const list = own(entry, "regions");
  if (Array.isArray(list) && list.length > 0) {
    const first = list[0];
    return isObj(first) ? str(own(first, "name")) : str(first);
  }
  return "";
}

function readEntry(name: string, entry: unknown, secure: unknown): CloudsEntry {
  if (!isObj(entry)) return { name, ok: false, message: "This entry is not a cloud." };
  const sec = isObj(secure) ? secure : {};
  const auth: Obj = { ...(isObj(own(entry, "auth")) ? (own(entry, "auth") as Obj) : {}), ...(isObj(own(sec, "auth")) ? (own(sec, "auth") as Obj) : {}) };
  const merged: Obj = { ...entry, ...sec, auth };
  const authType = str(own(merged, "auth_type")).toLowerCase().replace(/[-_]/g, "");

  const id = str(own(auth, APP_CRED_ID));
  const secret = str(own(auth, APP_CRED_SECRET));
  const appCred =
    authType === "v3applicationcredential" ||
    authType === "applicationcredential" ||
    !!id ||
    !!secret ||
    !!str(own(auth, "application_credential_name"));

  if (!appCred) {
    if (authType === "" || authType === "password" || authType === "v3password") {
      return {
        name,
        ok: false,
        message: `This entry signs in with a user name and password. Clutter needs an application credential instead, so it never holds your password. ${APP_CRED_HELP}`,
      };
    }
    return { name, ok: false, message: `This entry uses a sign-in method Clutter does not take. Clutter needs an application credential. ${APP_CRED_HELP}` };
  }
  if (!id && str(own(auth, "application_credential_name"))) {
    return {
      name,
      ok: false,
      message: `This entry names its application credential instead of giving its ID. ${APP_CRED_HELP} That file holds the ID.`,
    };
  }

  const values: Record<string, string> = {};
  const missing: string[] = [];
  const rawUrl = str(own(auth, AUTH_URL));
  if (rawUrl) {
    const n = normalizeAuthUrl(rawUrl, str(own(merged, "identity_api_version")));
    if (!n.ok) return { name, ok: false, message: n.message };
    values[AUTH_URL] = n.url;
  } else {
    missing.push("auth URL");
  }
  if (id) values[APP_CRED_ID] = id;
  else missing.push("application credential ID");
  if (secret) values[APP_CRED_SECRET] = secret;
  else missing.push("application credential secret");
  for (const v of Object.values(values)) {
    if (v.length > MAX_VALUE) return { name, ok: false, message: "A value in this entry is too long to be a credential." };
  }
  const region = entryRegion(merged);
  if (!region) missing.push("region");
  return { name, ok: true, values, region, missing };
}

// readCloudsFiles reads a clouds.yaml and, when given, its secure.yaml (whose
// values win, as the OpenStack tools merge them). It lists every cloud entry,
// each either ready to fill the form or refused with a reason.
export function readCloudsFiles(cloudsText: string, secureText?: string | null): CloudsFile {
  const clouds = readYaml(cloudsText, "clouds.yaml");
  if (!clouds.ok) return clouds;
  let secureClouds: Obj = {};
  if (secureText != null && secureText.trim() !== "") {
    const secure = readYaml(secureText, "secure.yaml");
    if (!secure.ok) return secure;
    secureClouds = own(secure.doc, "clouds") as Obj;
  }
  const all = own(clouds.doc, "clouds") as Obj;
  const names = Object.keys(all);
  if (names.length === 0) return { ok: false, message: "The clouds.yaml file lists no clouds." };
  return { ok: true, entries: names.map((n) => readEntry(n, own(all, n), own(secureClouds, n))) };
}

// splitCloudsFiles picks the clouds.yaml and the secure.yaml out of the files
// a user added, by name: a file whose name has "secure" in it is secure.yaml.
export function splitCloudsFiles<F extends { name: string }>(files: F[]): { ok: true; clouds: F; secure: F | null } | { ok: false; message: string } {
  const secure = files.filter((f) => /secure/i.test(f.name));
  const clouds = files.filter((f) => !/secure/i.test(f.name));
  if (clouds.length !== 1 || secure.length > 1) {
    return { ok: false, message: "Add one clouds.yaml, and its secure.yaml if your cloud gave you one." };
  }
  return { ok: true, clouds: clouds[0], secure: secure[0] ?? null };
}
