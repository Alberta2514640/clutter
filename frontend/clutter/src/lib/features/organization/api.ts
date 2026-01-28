// lib/features/organization/api.ts
import type { ApiEnvelope, CreateOrganizationInput, Organization, UpdateOrganizationInput } from "./types";

const API_BASE = process.env.NEXT_PUBLIC_API_ENDPOINT;

async function apiFetch<T>(path: string, token: string, init: RequestInit = {}): Promise<T> {
  if (!API_BASE) throw new Error("API_ENDPOINT is not set");

  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      Authorization: `${token}`,
      ...(init.headers ?? {}),
    },
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(text || `Request failed: ${res.status}`);
  }

  return (await res.json()) as T;
}

export const organizationApi = {
  // GET /organization
  // supports either {data: {...}} or {data: [{...}, ...]}
  list: async (token: string): Promise<Organization[]> => {
    const json = await apiFetch<ApiEnvelope<Organization | Organization[]>>("/organization", token, { method: "GET" });

    return Array.isArray(json.data) ? json.data : [json.data];
  },

  // POST /organization
  create: async (token: string, input: CreateOrganizationInput): Promise<Organization> => {
    const json = await apiFetch<ApiEnvelope<Organization>>("/organization", token, {
      method: "POST",
      body: JSON.stringify(input),
    });

    return json.data;
  },

  // PUT /organization/:organizationId
  update: async (token: string, organizationId: string, input: UpdateOrganizationInput): Promise<Organization> => {
    const json = await apiFetch<ApiEnvelope<Organization>>(`/organization/${organizationId}`, token, {
      method: "PUT",
      body: JSON.stringify(input),
    });

    return json.data;
  },

  // DELETE /organization/:organizationId
  delete: async (token: string, organizationId: string): Promise<void> => {
    await apiFetch<ApiEnvelope<unknown>>(`/organization/${organizationId}`, token, {
      method: "DELETE",
    });
  },
};
