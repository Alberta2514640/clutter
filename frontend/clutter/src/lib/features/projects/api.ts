// lib/features/projects/api.ts
import type { Project } from "./types";

import type { ApiEnvelope } from "@/lib/features/organization/types"; // or move ApiEnvelope to a shared /lib/api/types

const API_BASE = process.env.NEXT_PUBLIC_API_ENDPOINT;

async function apiFetch<T>(path: string, token: string, init: RequestInit = {}): Promise<T> {
  if (!API_BASE) throw new Error("API_ENDPOINT is not set");
  if (!token) throw new Error("Missing auth token");

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

export const projectsApi = {
  // GET /project?organizationId=...

  listByOrganization: async (token: string, organizationId: string): Promise<Project[]> => {
    if (!organizationId) throw new Error("Missing organizationId");

    const json = await apiFetch<ApiEnvelope<Project | Project[]> | { projects: Project[] }>(`/project?organizationId=${encodeURIComponent(organizationId)}`, token, { method: "GET" });

    // handle both envelope + legacy shape
    if ("projects" in json) return json.projects;

    return Array.isArray(json.data) ? json.data : [json.data];
  },

  // POST /project
  create: async (token: string, input: { organizationId: string; name: string; description?: string }): Promise<Project> => {
    if (!input.organizationId) throw new Error("Missing organizationId");
    if (!input.name?.trim()) throw new Error("Missing project name");

    const json = await apiFetch<ApiEnvelope<Project>>(`/project`, token, {
      method: "POST",
      body: JSON.stringify({
        organizationId: input.organizationId,
        name: input.name,
        description: input.description?.trim() ?? "",
      }),
    });

    return json.data;
  },

  // GET /project/:projectId  (adjust path if your backend is /projects/:id)
  getById: async (token: string, projectId: string): Promise<Project> => {
    if (!projectId) throw new Error("Missing projectId");

    const json = await apiFetch<ApiEnvelope<Project>>(`/project/${projectId}`, token, { method: "GET" });

    return json.data;
  },

  // PUT /project/:projectId
  update: async (token: string, projectId: string, input: Partial<Project>): Promise<Project> => {
    if (!projectId) throw new Error("Missing projectId");

    const json = await apiFetch<ApiEnvelope<Project>>(`/project/${projectId}`, token, {
      method: "PUT",
      body: JSON.stringify(input),
    });

    return json.data;
  },

  // DELETE /project/:projectId
  delete: async (token: string, projectId: string): Promise<void> => {
    if (!projectId) throw new Error("Missing projectId");

    await apiFetch<ApiEnvelope<unknown>>(`/project/${projectId}`, token, { method: "DELETE" });
  },
};
