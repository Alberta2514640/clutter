// The one place that knows how A's API is called with the user's token and
// what an expired session looks like. B's client (src/lib/adapterApi.ts) has its own
// fetch; its errors carry the same `status`, so isSessionExpired reads both.

import { clearStoredSession } from "./api";

const API_BASE = process.env.NEXT_PUBLIC_API_ENDPOINT;

// Where an expired session lands. The login page reads `session=expired`.
export const SESSION_EXPIRED_PATH = "/login?session=expired";
export const SESSION_EXPIRED_NOTICE = "Your session expired. Sign in again.";

// ApiError is a refused call to A's API. Its message stays the raw response
// body, because A's editor parses it for the diagram's own problems; status
// lets callers pick a message for the user without reading that body.
export class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

// apiErrorMessage returns the reason A's API gave for refusing a call: the
// body's `error` (the specific reason) or else its `message`. It returns null
// for any other error, or a body that is not A's JSON.
export function apiErrorMessage(err: unknown): string | null {
  if (!(err instanceof ApiError)) return null;
  try {
    const parsed = JSON.parse(err.message) as { error?: unknown; message?: unknown };
    if (typeof parsed.error === "string" && parsed.error.trim()) return parsed.error.trim();
    if (typeof parsed.message === "string" && parsed.message.trim()) return parsed.message.trim();
  } catch {
    // Not JSON: the request itself failed.
  }
  return null;
}

export async function apiFetch<T>(path: string, token: string, init: RequestInit = {}): Promise<T> {
  if (!API_BASE) throw new ApiError(0, "NEXT_PUBLIC_API_ENDPOINT is not set");

  let res: Response;
  try {
    res = await fetch(`${API_BASE}${path}`, {
      ...init,
      headers: {
        "Content-Type": "application/json",
        Authorization: `${token}`,
        ...(init.headers ?? {}),
      },
    });
  } catch {
    throw new ApiError(0, "Clutter could not reach its server. Check your connection and try again.");
  }

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new ApiError(res.status, text || `Request failed: ${res.status}`);
  }

  return (await res.json()) as T;
}

// API Gateway answers a token its authorizer refuses (expired or not valid)
// with 403 and this text. A handler's own 403 (not a member) never uses it.
const GATEWAY_DENY = "User is not authorized to access this resource";

// isSessionExpired reports whether an error means the token no longer works:
// B answers 401, A's gateway 401 without a token and 403 with GATEWAY_DENY.
// The text check also covers A's features that throw plain errors.
export function isSessionExpired(err: unknown): boolean {
  const status = (err as { status?: unknown } | null)?.status;
  if (status === 401) return true;
  const message = err instanceof Error ? err.message : "";
  return message.includes(GATEWAY_DENY);
}

// isConflict reports A's 409: a duplicate name, or a delete the database
// refused because live deployments remain.
export function isConflict(err: unknown): boolean {
  return err instanceof ApiError && err.status === 409;
}

// expireSession clears the stored token and sends the browser to the login
// page with the expired-session notice. A full page load drops every cached
// query, so nothing from the old session is shown again.
export function expireSession(): void {
  if (typeof window === "undefined") return;
  clearStoredSession();
  if (window.location.pathname === "/login" && window.location.search.includes("session=expired")) return;
  window.location.replace(SESSION_EXPIRED_PATH);
}
