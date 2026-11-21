// lib/features/projects/api.ts
import { apiFetch } from "../user/session";
import type { ApiEnvelope, CreateProjectInput, Project, UpdateProjectInput } from "./types";

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

  // GET /project?organizationId=...&projectId=...
  getById: async (token: string, organizationId: string, projectId: string): Promise<Project> => {
    const json = await apiFetch<ApiEnvelope<Project>>(
      `/project?organizationId=${encodeURIComponent(organizationId)}&projectId=${encodeURIComponent(projectId)}`,
      token,
      { method: "GET" }
    );

    return json.data;
  },

  // PUT /project?projectId=...
  update: async (token: string, projectId: string, input: UpdateProjectInput): Promise<Project> => {
    const json = await apiFetch<ApiEnvelope<Project>>(`/project?projectId=${encodeURIComponent(projectId)}`, token, {
      method: "PUT",
      body: JSON.stringify(input),
    });

    return json.data;
  },

  // DELETE /project?organizationId=...&projectId=...
  delete: async (token: string, organizationId: string, projectId: string): Promise<void> => {
    await apiFetch<ApiEnvelope<unknown>>(
      `/project?organizationId=${encodeURIComponent(organizationId)}&projectId=${encodeURIComponent(projectId)}`,
      token,
      { method: "DELETE" }
    );
  },
};
