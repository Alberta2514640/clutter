import type { Project } from "./types";

const API_BASE = process.env.NEXT_PUBLIC_API_ENDPOINT;

export const projectsApi = {
  listByToken: async (token: string, organizationId: string): Promise<Project[]> => {
    if (!token) {
      throw new Error("Missing auth token");
    }

    const res = await fetch(`${API_BASE}/project?organizationId=${organizationId}`, {
      method: "GET",
      headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
    });
    if (!res.ok) {
      throw new Error(`Error fetching projects: ${res.statusText}`);
    }
    const data = await res.json();
    return data.projects as Project[];
  },

  create: async (token: string, payload: { organizationId: string; name: string; description?: string }): Promise<Project> => {
    if (!token) throw new Error("Missing auth token");
    if (!payload.organizationId) throw new Error("Missing organizationId");
    if (!payload.name?.trim()) throw new Error("Missing project name");

    const res = await fetch(`${API_BASE}/project`, {
      method: "POST",
      headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
      body: JSON.stringify({
        organizationId: payload.organizationId,
        name: payload.name.trim(),
        description: payload.description?.trim() ?? "",
      }),
    });

    if (!res.ok) {
      const text = await res.text().catch(() => "");
      throw new Error(`Error creating project: ${res.status} ${res.statusText} ${text}`);
    }

    const data = await res.json();

    const created = (data?.project ?? data) as Project;

    if (!created?.projectId) {
      console.warn("Create project response missing projectId:", data);
    }

    return created;
  },

  getById: async (token: string, projectId: string): Promise<Project> => {
    const res = await fetch(`${API_BASE}/projects/${projectId}`, {
      method: "GET",
      headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
    });

    if (!res.ok) {
      throw new Error(`Error fetching project: ${res.status} ${res.statusText}`);
    }

    const data = await res.json();

    return data as Project;
  },

  update: async (projectId: string, data: Partial<Project>, token: string): Promise<Project> => {
    const res = await fetch(`${API_BASE}/projects/${projectId}`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(data),
    });

    if (!res.ok) {
      throw new Error(`Error updating project: ${res.status} ${res.statusText}`);
    }

    const updatedProject = await res.json();
    return updatedProject as Project;
  },

  delete: async (projectId: string, token: string): Promise<void> => {
    const res = await fetch(`${API_BASE}/projects/${projectId}`, {
      method: "DELETE",
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });

    if (!res.ok) {
      throw new Error(`Error deleting project: ${res.status} ${res.statusText}`);
    }
  },
};
