// lib/features/projects/api.ts
import type {
  ApiEnvelope,
  CreateProjectInput,
  GetTerraformLogsInput,
  Project,
  TerraformLogsResponse,
  UpdateProjectInput,
} from "./types";

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

  // GET /terraform-engine/logs?orgId=...&projId=...&diagramId=...
  getTerraformLogs: async (token: string, input: GetTerraformLogsInput): Promise<TerraformLogsResponse> => {
    const qs = new URLSearchParams({
      orgId: input.orgId,
      projId: input.projId,
      diagramId: input.diagramId,
    });

    const json = await apiFetch<ApiEnvelope<TerraformLogsResponse> | TerraformLogsResponse>(
      `/terraform-engine/logs?${qs.toString()}`,
      token,
      { method: "GET" }
    );

    if (json && typeof json === "object" && "data" in json) {
      return json.data as TerraformLogsResponse;
    }

    return json as TerraformLogsResponse;
  },
};
