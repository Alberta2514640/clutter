// Checks if a user may open a diagram. This runs for every connection.
// We do not decide access ourselves. We call A's GET /diagram with the
// user's own token, and A decides.
// So this service needs no JWT secret and no database password.
// Never log the token.

import { CLOSE, type CloseCode } from "./codes.ts";
import type { Config } from "./config.ts";

// Name of the login cookie. The website sets it after sign-in.
const COOKIE_NAME = "clutter_token";
const BLOCKS_ENGINE = "blocks";
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

export interface AccessRequest {
  cookieHeader: string | undefined;
  diagramId: string;
  projectId: string | null;
}

export type Access =
  | { ok: true; userId: string; diagramName: string; document: Record<string, unknown> }
  | { ok: false; code: CloseCode; reason: string };

const refuse = (code: CloseCode, reason: string): Access => ({ ok: false, code, reason });

const isRecord = (v: unknown): v is Record<string, unknown> => !!v && typeof v === "object" && !Array.isArray(v);

// Get the login token from the Cookie header. Returns null if it is missing.
export function readToken(cookieHeader: string | undefined): string | null {
  if (!cookieHeader) return null;
  for (const part of cookieHeader.split(";")) {
    const eq = part.indexOf("=");
    if (eq < 0 || part.slice(0, eq).trim() !== COOKIE_NAME) continue;
    try {
      return decodeURIComponent(part.slice(eq + 1).trim()) || null;
    } catch {
      return null;
    }
  }
  return null;
}

// Read the user id ("sub") from the token.
// This does not verify the token. Only call it after A accepted the token.
function userIdOf(token: string): string | null {
  const payload = token.split(".")[1];
  if (!payload) return null;
  try {
    const claims: unknown = JSON.parse(Buffer.from(payload, "base64url").toString("utf8"));
    if (isRecord(claims) && typeof claims.sub === "string" && UUID.test(claims.sub)) return claims.sub;
  } catch {
    // The token is not valid JSON. Fall through and return null.
  }
  return null;
}

// Decide if this connection is allowed.
// Returns the saved diagram and the user id, or a close code.
export async function checkAccess(req: AccessRequest, cfg: Config, fetchFn: typeof fetch = fetch): Promise<Access> {
  if (!req.projectId) return refuse(CLOSE.badRequest, "projectId is required");
  // A returns 404 for an id that is not a UUID. We do the same and skip the call.
  if (!UUID.test(req.diagramId) || !UUID.test(req.projectId)) return refuse(CLOSE.notFound, "not found");

  const token = readToken(req.cookieHeader);
  if (!token) return refuse(CLOSE.unauthorized, "not signed in");

  const url = new URL("/diagram", cfg.apiUrl);
  url.searchParams.set("projectId", req.projectId);
  url.searchParams.set("diagramId", req.diagramId);

  let status: number;
  let body: unknown;
  try {
    // A wants the raw token, without "Bearer" in front.
    const res = await fetchFn(url, { headers: { Authorization: token }, signal: AbortSignal.timeout(cfg.accessTimeoutMs) });
    status = res.status;
    body = status === 200 ? await res.json() : null;
  } catch {
    return refuse(CLOSE.upstream, "could not reach the API");
  }

  // 401 or 403 means the token is missing, bad or expired.
  // A user who is not a member gets 404, not 403.
  if (status === 401 || status === 403) return refuse(CLOSE.unauthorized, "not signed in");
  if (status === 404) return refuse(CLOSE.notFound, "not found");
  if (status === 400) return refuse(CLOSE.badRequest, "bad request");
  if (status !== 200) return refuse(CLOSE.upstream, "the API failed");

  const diagram = isRecord(body) ? body.data : null;
  if (!isRecord(diagram) || diagram.id !== req.diagramId) return refuse(CLOSE.upstream, "the API sent an unexpected answer");
  // Only blocks diagrams can be shared live. Old AWS diagrams are refused.
  const document = diagram.data;
  if (!isRecord(document) || document.engine !== BLOCKS_ENGINE) return refuse(CLOSE.badRequest, "not a building-block diagram");

  const userId = userIdOf(token);
  if (!userId) return refuse(CLOSE.unauthorized, "not signed in");

  return { ok: true, userId, diagramName: typeof diagram.name === "string" ? diagram.name : "", document };
}
