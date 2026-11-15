import { describe, expect, it } from "vitest";
import { checkAccess, readToken, type AccessRequest } from "../src/access.ts";
import { CLOSE } from "../src/codes.ts";
import { loadConfig } from "../src/config.ts";

const cfg = loadConfig({});
const USER = "11111111-1111-4111-8111-111111111111";
const PROJECT = "22222222-2222-4222-8222-222222222222";
const DIAGRAM = "33333333-3333-4333-8333-333333333333";

// Make a fake token with the given claims.
// The service does not check the signature, so any signature works here.
function tokenFor(claims: Record<string, unknown>): string {
  return `e30.${Buffer.from(JSON.stringify(claims)).toString("base64url")}.sig`;
}
const TOKEN = tokenFor({ sub: USER, email: "a@clutter.localhost" });

const request = (over: Partial<AccessRequest> = {}): AccessRequest => ({
  cookieHeader: `other=1; clutter_token=${encodeURIComponent(TOKEN)}`,
  diagramId: DIAGRAM,
  projectId: PROJECT,
  ...over,
});

const blocks = { engine: "blocks", nodes: [], edges: [] };
const okBody = (data: unknown = blocks) => ({ message: "ok", data: { id: DIAGRAM, name: "Demo", data } });

// A fake of A's GET /diagram. It saves each call so the tests can check it.
function fakeApi(status: number, body: unknown = null) {
  const calls: { url: string; authorization: string | null }[] = [];
  const fetchFn = (async (input: unknown, init?: RequestInit) => {
    calls.push({ url: String(input), authorization: new Headers(init?.headers).get("Authorization") });
    return new Response(body === null ? null : JSON.stringify(body), { status });
  }) as typeof fetch;
  return { calls, fetchFn };
}

describe("readToken", () => {
  it("finds the cookie among others and decodes it", () => {
    expect(readToken("a=b; clutter_token=x%2Ey; c=d")).toBe("x.y");
  });
  it.each([undefined, "", "a=b", "clutter_token=", "clutter_token=%E0%A4%A"])("returns null for %j", (header) => {
    expect(readToken(header)).toBeNull();
  });
});

describe("checkAccess", () => {
  it("lets a member in and returns the saved document", async () => {
    const api = fakeApi(200, okBody());
    const got = await checkAccess(request(), cfg, api.fetchFn);
    expect(got).toEqual({ ok: true, userId: USER, diagramName: "Demo", document: blocks });
    expect(api.calls).toEqual([
      { url: `http://localhost:8787/diagram?projectId=${PROJECT}&diagramId=${DIAGRAM}`, authorization: TOKEN },
    ]);
  });

  it("refuses a missing projectId without calling the API", async () => {
    const api = fakeApi(200, okBody());
    const got = await checkAccess(request({ projectId: null }), cfg, api.fetchFn);
    expect(got).toMatchObject({ ok: false, code: CLOSE.badRequest });
    expect(api.calls).toHaveLength(0);
  });

  it.each([
    ["diagramId", { diagramId: "not-a-uuid" }],
    ["projectId", { projectId: `${PROJECT}&diagramId=x` }],
  ])("answers not found for a %s that is not a UUID, without calling the API", async (_name, over) => {
    const api = fakeApi(200, okBody());
    const got = await checkAccess(request(over), cfg, api.fetchFn);
    expect(got).toMatchObject({ ok: false, code: CLOSE.notFound });
    expect(api.calls).toHaveLength(0);
  });

  it("refuses a request with no cookie without calling the API", async () => {
    const api = fakeApi(200, okBody());
    const got = await checkAccess(request({ cookieHeader: undefined }), cfg, api.fetchFn);
    expect(got).toMatchObject({ ok: false, code: CLOSE.unauthorized });
    expect(api.calls).toHaveLength(0);
  });

  it.each([
    [401, CLOSE.unauthorized],
    [403, CLOSE.unauthorized],
    [404, CLOSE.notFound],
    [400, CLOSE.badRequest],
    [500, CLOSE.upstream],
    [502, CLOSE.upstream],
  ])("maps the API's %i to close code %i", async (status, code) => {
    const got = await checkAccess(request(), cfg, fakeApi(status).fetchFn);
    expect(got).toMatchObject({ ok: false, code });
  });

  it("answers upstream when the API cannot be reached", async () => {
    const fetchFn = (async () => {
      throw new TypeError("fetch failed");
    }) as typeof fetch;
    expect(await checkAccess(request(), cfg, fetchFn)).toMatchObject({ ok: false, code: CLOSE.upstream });
  });

  it("answers upstream when the API does not answer in time", async () => {
    const fetchFn = ((_input: unknown, init?: RequestInit) =>
      new Promise((_resolve, reject) => {
        init?.signal?.addEventListener("abort", () => reject(init.signal?.reason));
      })) as typeof fetch;
    const got = await checkAccess(request(), { ...cfg, accessTimeoutMs: 20 }, fetchFn);
    expect(got).toMatchObject({ ok: false, code: CLOSE.upstream });
  });

  it.each([
    ["an AWS diagram (no engine)", { nodes: [], edges: [] }],
    ["an unknown engine", { engine: "other", nodes: [], edges: [] }],
    ["no document", null],
  ])("refuses %s as a bad request", async (_name, data) => {
    const got = await checkAccess(request(), cfg, fakeApi(200, okBody(data)).fetchFn);
    expect(got).toMatchObject({ ok: false, code: CLOSE.badRequest });
  });

  it("answers upstream when the API returns another diagram or a body it cannot read", async () => {
    const other = { message: "ok", data: { id: PROJECT, name: "x", data: blocks } };
    expect(await checkAccess(request(), cfg, fakeApi(200, other).fetchFn)).toMatchObject({ ok: false, code: CLOSE.upstream });
    expect(await checkAccess(request(), cfg, fakeApi(200, []).fetchFn)).toMatchObject({ ok: false, code: CLOSE.upstream });
  });

  it.each([
    ["no sub", tokenFor({ email: "a@clutter.localhost" })],
    ["a sub that is not a UUID", tokenFor({ sub: "admin" })],
    ["not a JWT", "opaque-token"],
  ])("refuses a token with %s even when the API accepts it", async (_name, token) => {
    const api = fakeApi(200, okBody());
    const got = await checkAccess(request({ cookieHeader: `clutter_token=${token}` }), cfg, api.fetchFn);
    expect(got).toMatchObject({ ok: false, code: CLOSE.unauthorized });
  });

  it("never puts the token in a refusal", async () => {
    for (const status of [401, 404, 500]) {
      const got = await checkAccess(request(), cfg, fakeApi(status).fetchFn);
      expect(JSON.stringify(got)).not.toContain(TOKEN);
    }
  });
});
