export async function api<T>(url: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(url, {
    credentials: "include",
    headers: { "Content-Type": "application/json", ...(init.headers || {}) },
    ...init,
  });
  if (!res.ok) throw Object.assign(new Error(res.statusText), { status: res.status, body: await res.text() });
  return res.status === 204 ? (undefined as T) : ((await res.json()) as T);
}
export const apiWithIfMatch = <B, R>(url: string, body: B, etag: string) => api<R>(url, { method: "PATCH", body: JSON.stringify(body), headers: { "If-Match": etag } });
