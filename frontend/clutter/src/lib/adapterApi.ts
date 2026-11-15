// The client for B API (backend/adapter), the /v1 service beside A's API. It
// sends the same token A's features send, and reads B's one error envelope:
// {"error": {"code", "message", "problems"?}} (backend/adapter/internal/api).

import type { Problem } from "@/lib/features/deployments/types";

// The service's address, read once here at build time
// (NEXT_PUBLIC_ADAPTER_API_ENDPOINT). Every B call, fetch or XMLHttpRequest,
// uses it and the messages and parser below.
export const ADAPTER_API_BASE = process.env.NEXT_PUBLIC_ADAPTER_API_ENDPOINT;

// The one sentence every screen shows when the website was built without the
// service's address. No screen words this fact its own way.
export const NOT_CONFIGURED_MESSAGE = "Deploying is not set up on this website: it was built without the address of Clutter's deployment service.";
export const UNREACHABLE_MESSAGE = "Clutter could not reach its deployment service. Check your connection and try again.";

// AdapterApiError is a refused or failed call to B. Its message is B's own,
// which is written for the user; problems name the blocks or fields at fault.
export class AdapterApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly problems: Problem[];

  constructor(status: number, code: string, message: string, problems: Problem[] = []) {
    super(message);
    this.name = "AdapterApiError";
    this.status = status;
    this.code = code;
    this.problems = problems;
  }
}

export const adapterApiConfigured = !!ADAPTER_API_BASE;

async function request(path: string, token: string, init: RequestInit): Promise<Response> {
  if (!ADAPTER_API_BASE) throw new AdapterApiError(0, "not_configured", NOT_CONFIGURED_MESSAGE);

  let res: Response;
  try {
    res = await fetch(`${ADAPTER_API_BASE}${path}`, {
      ...init,
      headers: {
        ...(init.body !== undefined ? { "Content-Type": "application/json" } : {}),
        Authorization: token,
        ...(init.headers ?? {}),
      },
    });
  } catch {
    throw new AdapterApiError(0, "unreachable", UNREACHABLE_MESSAGE);
  }
  if (!res.ok) throw errorFromBody(res.status, await res.text().catch(() => ""), `The request failed (${res.status}).`);
  return res;
}

// errorFromBody reads B's error envelope from a refused answer's body, with
// its problems; a body that is not the envelope gives the fallback message.
export function errorFromBody(status: number, body: string, fallback: string): AdapterApiError {
  try {
    const parsed = JSON.parse(body) as { error?: { code?: unknown; message?: unknown; problems?: unknown } };
    const e = parsed?.error;
    if (e && typeof e.message === "string") {
      return new AdapterApiError(status, typeof e.code === "string" ? e.code : "error", e.message, Array.isArray(e.problems) ? (e.problems as Problem[]) : []);
    }
  } catch {
    // Not B's envelope: fall through to a plain message.
  }
  return new AdapterApiError(status, "error", fallback);
}

export async function adapterFetch<T>(path: string, token: string, init: RequestInit = {}): Promise<T> {
  const res = await request(path, token, init);
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

export async function adapterFetchText(path: string, token: string): Promise<string> {
  const res = await request(path, token, { method: "GET" });
  return res.text();
}

// adapterErrorMessage turns any error from a B call into one line for the user.
export function adapterErrorMessage(err: unknown): string {
  if (err instanceof AdapterApiError) return err.message;
  if (err instanceof Error && err.message) return err.message;
  return "Something went wrong. Try again.";
}

// An id segment for a path. B's ids are UUIDs or connector ids; encoding keeps
// any other value from changing the path.
export const seg = (id: string) => encodeURIComponent(id);
