// lib/features/projects/api.ts
import type { ApiEnvelope, CreateProjectInput, Project, UpdateProjectInput } from "./types";

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

export const projectsApi = {
  // GET /project?organizationId=...
  listByOrganization: async (token: string, organizationId: string): Promise<Project[]> => {
    const json = await apiFetch<ApiEnvelope<Project | Project[]> | { projects: Project[] }>(
      `/project?organizationId=${encodeURIComponent(organizationId)}`,
      token,
      { method: "GET" }
    );
    // handle both envelope + legacy shape
    if ("projects" in json) return json.projects;
    return Array.isArray(json.data) ? json.data : [json.data];
  },

  // POST /project
  create: async (token: string, input: CreateProjectInput): Promise<Project> => {
    const json = await apiFetch<ApiEnvelope<Project>>("/project", token, {
      method: "POST",
      body: JSON.stringify(input),
    });
    return json.data;
  },

  // GET /project/:projectId
  getById: async (token: string, projectId: string): Promise<Project> => {
    const json = await apiFetch<ApiEnvelope<Project>>(`/project/${projectId}`, token, { method: "GET" });
    return json.data;
  },

  // PUT /project/:projectId
  update: async (token: string, projectId: string, input: UpdateProjectInput): Promise<Project> => {
    const json = await apiFetch<ApiEnvelope<Project>>(`/project/${projectId}`, token, {
      method: "PUT",
      body: JSON.stringify(input),
    });
    return json.data;
  },

  // DELETE /project/:projectId
  delete: async (token: string, projectId: string): Promise<void> => {
    await apiFetch<ApiEnvelope<unknown>>(`/project/${projectId}`, token, { method: "DELETE" });
  },
};
